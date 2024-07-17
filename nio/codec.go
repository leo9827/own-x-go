package nio

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"encoding/json"
	"io"
)

type Codec interface {
	Encode(msg *Msg) *bytes.Buffer
	Decode(msg *Msg, reader *bytes.Buffer) bool
}

type Msg struct {
	flag   uint8       // msg type flag
	body   interface{} // msg body
	length uint32      // msg length
	data   []byte      // msg byte data
}

func (msg *Msg) Flag() uint8 {
	return msg.flag
}

func (msg *Msg) Body() interface{} {
	return msg.body
}

func (msg *Msg) Length() uint32 {
	return msg.length
}

func (msg *Msg) Data() []byte {
	return msg.data
}

func (msg *Msg) SetLength(length uint32) {
	msg.length = length
}

func (msg *Msg) SetData(data []byte) {
	msg.data = data
}

func CreateMsg(flag uint8, body interface{}) *Msg {
	return &Msg{
		flag: flag,
		body: body,
	}
}

type defaultCodec struct {
	gzip bool
}

func getDefaultCodec(gzip bool) *defaultCodec {
	return &defaultCodec{gzip: gzip}
}

func (c *defaultCodec) Encode(msg *Msg) *bytes.Buffer {
	buf := new(bytes.Buffer)
	var bodyBytes []byte
	if msg.body == nil {
		msg.length = 1
	} else {
		bodyBytes = ToByte(msg.body)
		if c.gzip {
			bodyBytes, _ = Gzip(bodyBytes)
		}
		msg.length = uint32(len(bodyBytes) + 1)
	}
	binary.Write(buf, binary.BigEndian, msg.length)
	binary.Write(buf, binary.BigEndian, msg.flag)
	if len(bodyBytes) > 0 {
		buf.Write(bodyBytes)
	}
	return buf
}

func (c *defaultCodec) Decode(msg *Msg, reader *bytes.Buffer) bool {
	if msg.length == 0 {
		if reader.Len() >= 5 {
			var length uint32
			binary.Read(reader, binary.BigEndian, &length)
			msg.length = length

			var flag uint8
			binary.Read(reader, binary.BigEndian, &flag)
			msg.flag = flag
		} else {
			return false
		}
	}
	if msg.length == 1 {
		return true
	}
	if msg.length > 1 && msg.data == nil {
		if reader.Len() >= int(msg.length-1) {
			bodyBytes := make([]byte, msg.length-1)
			reader.Read(bodyBytes)
			msg.data = bodyBytes
		} else {
			return false
		}
	}
	if c.gzip {
		msg.data, _ = UnGzip(msg.data)
	}
	return true
}

func Gzip(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return []byte{}, nil
	}
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	defer gz.Close()
	if _, err := gz.Write(data); err != nil {
		return nil, err
	}
	if err := gz.Flush(); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func UnGzip(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return []byte{}, nil
	}
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

func Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func ToByte(v interface{}) []byte {
	switch t := v.(type) {
	case string:
		return []byte(t)
	case []byte:
		return v.([]byte)
	}
	json, _ := Marshal(v)
	return json
}
