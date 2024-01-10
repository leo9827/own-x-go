package csv

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"os"
	"strconv"
)

type Spec struct {
	FileName string          // 文件名称
	Titles   []string        // 每列的标题
	Data     [][]interface{} // 每行数据
}

func Create(spec *Spec) error {
	if spec == nil {
		return errors.New("parameter missing")
	}
	if spec.FileName == "" {
		return errors.New("file name missing")
	}
	filename := spec.FileName
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer file.Close()
	csvWriter := csv.NewWriter(file)
	if len(spec.Titles) > 0 {
		err = csvWriter.Write(spec.Titles)
		if err != nil {
			return err
		}
	}
	if len(spec.Data) > 0 {
		var data = make([][]string, 0, len(spec.Data))
		for _, datarow := range spec.Data {
			row := make([]string, 0, len(datarow))
			for _, d := range datarow {
				row = append(row, toString(d))
			}
			data = append(data, row)
		}
		err = csvWriter.WriteAll(data)
		if err != nil {
			return err
		}
	}
	csvWriter.Flush()
	return nil
}

func toString(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case int:
		return strconv.Itoa(t)
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		if t {
			return "true"
		} else {
			return "false"
		}
	case []interface{}:
		// 数组类型
		b, _ := json.Marshal(t)
		return string(b)
	case map[string]interface{}:
		// map类型
		b, _ := json.Marshal(t)
		return string(b)
	case nil:
		return ""
	default:
		b, _ := json.Marshal(t)
		return string(b)
	}
}
