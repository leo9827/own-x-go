package nio

import (
	"bytes"
	"encoding/binary"
	"errors"
	"log"
	"net"
	"sync/atomic"
	"time"
)

var sessionId int64 = 0

type SocketConnector struct {
	socket
	session *Session
}

func CreateConnector(ip, port string, handler *Handler, codec Codec, ordered bool) (*SocketConnector, error) {
	if handler == nil {
		return nil, errors.New(ip + ":" + port + " Create socket connector failed. Handler is nil.")
	}
	connector := &SocketConnector{}
	connector.addr = ip + ":" + port
	connector.handler = handler
	connector.ordered = ordered
	connector.handler.codec = codec
	if codec == nil {
		connector.handler.codec = getDefaultCodec(false)
	}

	return connector, nil
}

func (socket *SocketConnector) Connect() error {

	log.Printf("Connect to server %s ...", socket.addr)

	connection, err := net.DialTimeout("tcp", socket.addr, 10*time.Second)
	if err != nil {
		return err
	}

	session := &Session{}
	session.id = atomic.AddInt64(&sessionId, 1)
	session.isClient = true
	session.codec = socket.handler.codec
	session.connector = socket
	session.connection = connection

	socket.session = session

	if sessionId, syncErr := socket.syncSessionId(); syncErr != nil {
		log.Printf(" %s Sync Session Id failed. Cause of %s\n", socket.addr, syncErr.Error())
		err := connection.Close()
		if err != nil {
			log.Printf(" %s Close connection failed. Cause of %s\n", socket.addr, err.Error())
		}
		return syncErr
	} else {
		session.id = sessionId
	}

	log.Printf("[%d] Connect to server %s success. Local %s", session.id, socket.addr, connection.LocalAddr())

	if socket.handler.OnSessionConnected != nil {
		func() {
			defer OnError("")
			socket.handler.OnSessionConnected(session)
		}()
	}

	go socket.handler.readIo(&socket.socket, session)

	return nil
}

func (socket *SocketConnector) WriteMsg(msg *Msg) error {
	if err := socket.session.WriteMsg(msg); err != nil {
		return err
	}
	return nil
}

func (socket *SocketConnector) Write(flag uint8, data interface{}) error {
	if err := socket.session.Write(flag, data); err != nil {
		return err
	}
	return nil
}

func (socket *SocketConnector) Close() {
	log.Printf("[%d] Close socket connection %s.", socket.session.id, socket.addr)

	socket.closed = true

	// async call
	if socket.handler.OnSessionClosed != nil {
		go func() {
			defer OnError("")
			socket.handler.OnSessionClosed(socket.session)
		}()
	}

	// close socket
	for i := 1; i <= 3; i++ {
		if err := socket.session.connection.Close(); err != nil {
			log.Printf("[%d] Close socket connection %s error. %s", socket.session.id, socket.addr, err.Error())
			time.Sleep(500 * time.Millisecond)
		} else {
			return
		}
	}
}

func (socket *SocketConnector) syncSessionId() (int64, error) {
	defer OnError("Sync Session Id")
	firstData := bytes.NewBuffer([]byte{})
	firstDataLen := 8
	for {
		buf := make([]byte, firstDataLen)
		rlen, ioErr := socket.session.connection.Read(buf)
		if ioErr != nil {
			return 0, ioErr
		}
		if rlen > 0 {
			firstData.Write(buf[0:rlen])
		}
		firstDataLen = firstDataLen - rlen
		if firstDataLen == 0 {
			break
		}
	}
	var sid int64
	binary.Read(firstData, binary.BigEndian, &sid)
	return sid, nil
}
