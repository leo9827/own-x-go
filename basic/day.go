package basic

// 定义日期类型

import (
	"database/sql/driver"
	"encoding/xml"
	"time"
)

const (
	DayFormart = "2006-01-02"
)

type Day time.Time

func NewDay(str string) (Day, error) {
	t := Day{}
	var err error
	if str != "" && str != "null" {
		err = t.UnmarshalJSON([]byte(`"` + str + `"`))
	}
	return t, err
}

// UnmarshalJSON 时间类型反序列化
func (t *Day) UnmarshalJSON(data []byte) (err error) {
	str := string(data)
	if str == "\"\"" || str == "null" {
		return
	}
	now, err := time.ParseInLocation(`"`+DayFormart+`"`, str, time.Local)
	if err != nil {
		return err
	}
	*t = Day(now)
	return
}

// MarshalJSON 时间类型序列化
func (t Day) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return []byte("\"\""), nil
	}
	b := make([]byte, 0, len(DayFormart)+2)
	b = append(b, '"')
	b = t.Time().AppendFormat(b, DayFormart)
	b = append(b, '"')
	return b, nil
}

func (t Day) ToString() string {
	if t.IsZero() {
		return ""
	}
	return t.Time().Format(DayFormart)
}

func (t Day) Time() time.Time {
	tm := time.Time(t)
	if tm.IsZero() {
		return tm
	}
	return time.Date(tm.Year(), tm.Month(), tm.Day(), 0, 0, 0, 0, time.Local)
}

func (t Day) IsZero() bool {
	return t.Time().IsZero()
}

func (t Day) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	e.EncodeElement(t.ToString(), start)
	return nil
}

// 数据库字段反序列化
func (t *Day) Scan(val interface{}) (err error) {
	if val == nil {
		return
	}
	str := string(val.([]byte))
	if str == "\"\"" || str == "null" || str == "0000-00-00" || str == "0000-00-00 00:00:00" {
		return
	}
	if len(str) > 10 {
		str = str[:10]
	}
	now, err := time.ParseInLocation(DayFormart, str, time.Local)
	if err != nil {
		return err
	}
	*t = Day(now)
	return
}

// 数据库字段序列化
func (t Day) Value() (driver.Value, error) {
	if t.IsZero() {
		return nil, nil
	}
	return t.ToString(), nil
}

// 转换为time.Time，截去时分秒部分
func (t Day) ToTime() time.Time {
	return time.Date(t.Time().Year(), t.Time().Month(), t.Time().Day(), 0, 0, 0, 0, t.Time().Location())
}
func (t Day) FirstDayInMonth() Day {
	return Day(time.Date(t.Time().Year(), t.Time().Month(), 1, 0, 0, 0, 0, t.Time().Location()))
}
func (t Day) LastDayInMonth() Day {
	_tm := time.Date(t.Time().Year(), t.Time().Month(), 1, 0, 0, 0, 0, t.Time().Location())
	_tm = _tm.AddDate(0, 1, 0).AddDate(0, 0, -1)
	return Day(_tm)
}
