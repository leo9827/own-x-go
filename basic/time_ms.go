package basic

// 定义时间类型

// Time 和 TimeStandard 两个时间的区别
// Time
//     是不严格的，当将 json 字符串转换成 struct 结构体字段、或将数据库字段的值转换成 struct 结构体的字段时，不会返回错误
//     所以使用 Time 的场景一般都是调用外部接口时，由于不同系统返回的时间格式五花八门，不想因为时间格式错误而报接口错误，使用 Time 可将错误降级
// TimeStandard
//     是严格的，如果转换出错，会返回错误
//     使用 TimeStandard 的场景，一般是本系统对外暴露的接口参数，如果外部调用本系统接口传参，时间字段传参错应该返回错误，而不应该使用 Time 将错误降级

import (
	"database/sql/driver"
	"encoding/xml"
	"time"
)

type TimeMs time.Time

func NewTimsMs(str string) TimeMs {
	t := TimeMs{}
	if str != "" && str != "null" {
		t.UnmarshalJSON([]byte(`"` + str + `"`))
	}
	return t
}

// Json 时间类型反序列化
func (t *TimeMs) UnmarshalJSON(data []byte) (err error) {
	now, time_err := unmarshalJSONToJson(data)
	if time_err != nil {
		return time_err
	}
	if !now.IsZero() {
		now = now.Local()
		*t = TimeMs(now)
	}
	return
}

// Json 时间类型序列化
func (t TimeMs) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return []byte("\"\""), nil
	}
	b := make([]byte, 0, len(TimeFormart+".000")+2)
	b = append(b, '"')
	b = time.Time(t).AppendFormat(b, TimeFormart+".000")
	b = append(b, '"')
	return b, nil
}

func (t TimeMs) String() string {
	if t.IsZero() {
		return ""
	}
	return t.Time().Format(TimeFormart + ".000")
}

func (t TimeMs) UTCString() string {
	if t.IsZero() {
		return ""
	}
	return t.Time().UTC().Format("2006-01-02T15:04:05Z")
}

func (t TimeMs) Time() time.Time {
	return time.Time(t)
}

func (t TimeMs) IsZero() bool {
	return time.Time(t).IsZero()
}

func (t TimeMs) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	e.EncodeElement(t.String(), start)
	return nil
}

// 数据库字段反序列化
func (t *TimeMs) Scan(val interface{}) (err error) {
	if val == nil {
		return
	}
	str := string(val.([]byte))
	if str == "" || str == "\"\"" || str == "null" || str == "0000-00-00 00:00:00" {
		return
	}
	now, err := time.ParseInLocation(TimeFormart, str, time.Local)
	if err != nil {
		return err
	}
	*t = TimeMs(now)
	return
}

// 数据库字段序列化
func (t TimeMs) Value() (driver.Value, error) {
	if t.IsZero() {
		return nil, nil
	}
	return t.String(), nil
}
