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
	"fmt"
	"regexp"
	"strings"
	"time"
)

const (
	TimeFormart  = "2006-01-02 15:04:05"
	TimeFormartT = "2006-01-02T15:04:05"
)

type Time time.Time

func NewTime(str string) Time {
	t := Time{}
	if str != "" && str != "null" {
		t.UnmarshalJSON([]byte(`"` + str + `"`))
	}
	return t
}

// Json 时间类型反序列化
func (t *Time) UnmarshalJSON(data []byte) (err error) {
	now, time_err := unmarshalJSONToJson(data)
	if time_err != nil {
		return time_err
	}
	if !now.IsZero() {
		now = now.Local()
		*t = Time(now)
	}
	return
}

// Json 时间类型序列化
func (t Time) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return []byte("\"\""), nil
	}
	b := make([]byte, 0, len(TimeFormart)+2)
	b = append(b, '"')
	b = time.Time(t).AppendFormat(b, TimeFormart)
	b = append(b, '"')
	return b, nil
}

func (t Time) String() string {
	if t.IsZero() {
		return ""
	}
	return t.Time().Format(TimeFormart)
}

func (t Time) UTCString() string {
	if t.IsZero() {
		return ""
	}
	return t.Time().UTC().Format("2006-01-02T15:04:05Z")
}

func (t Time) Time() time.Time {
	return time.Time(t)
}

func (t Time) IsZero() bool {
	return time.Time(t).IsZero()
}

func (t Time) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	e.EncodeElement(t.String(), start)
	return nil
}

// 数据库字段反序列化
func (t *Time) Scan(val interface{}) (err error) {
	if val == nil {
		return
	}
	if _time, ok := val.(time.Time); ok {
		*t = Time(_time)
	} else if _, ok := val.([]byte); ok {
		str := string(val.([]byte))
		if str == "" || str == "\"\"" || str == "null" || str == "0000-00-00 00:00:00" {
			return
		}
		now, err := time.ParseInLocation(TimeFormart, str, time.Local)
		if err != nil {
			return err
		}
		*t = Time(now)
	} else {
		return fmt.Errorf("time type convert error. invalid value type")
	}
	return
}

// 数据库字段序列化
func (t Time) Value() (driver.Value, error) {
	if t.IsZero() {
		return `0000-00-00 00:00:00`, nil
	}
	return t.String(), nil
}

var (
	TimePattern1 *regexp.Regexp
	TimePattern2 *regexp.Regexp
	TimePattern3 *regexp.Regexp
	TimePattern4 *regexp.Regexp
	TimePattern5 *regexp.Regexp
)

// 解析时间字符串
func unmarshalJSONToJson(data []byte) (time.Time, error) {
	str := string(data)
	if str == "\"\"" || str == "null" {
		return time.Time{}, nil
	}
	loc := time.UTC
	format := TimeFormart
	if strings.Index(str, "T") == 11 {
		format = TimeFormartT
	}
	byteStr := []byte(str)
	if TimePattern1.Match(byteStr) {
		format = format + ".999999999 Z0700 CST"
	} else if TimePattern2.Match(byteStr) {
		format = format + "Z07:00"
	} else if TimePattern3.Match(byteStr) {
		format = format + ".999999999Z"
	} else if TimePattern4.Match(byteStr) {
		format = format + "Z"
	} else if TimePattern5.Match(byteStr) {
		format = format + ".999999999"
		loc = time.Local
	} else {
		loc = time.Local
	}
	return time.ParseInLocation(`"`+format+`"`, str, loc)
	//	pattern := regexp.MustCompile(end + `$`)
	//	return pattern.Match([]byte(s))
}

func init() {
	TimePattern1 = regexp.MustCompile("[.][0-9]+ [+][0-9]{4} CST\"$")
	TimePattern2 = regexp.MustCompile("[+][0-9]{2}:[0-9]{2}\"$")
	TimePattern3 = regexp.MustCompile("[.][0-9]+Z\"$")
	TimePattern4 = regexp.MustCompile("[:][0-9]{2}Z\"$")
	TimePattern5 = regexp.MustCompile("[.][0-9]+\"$")
}
