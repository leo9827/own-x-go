package csv

import (
	"encoding/csv"
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
	csvw := csv.NewWriter(file)
	if len(spec.Titles) > 0 {
		err = csvw.Write(spec.Titles)
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
		err = csvw.WriteAll(data)
		if err != nil {
			return err
		}
	}
	csvw.Flush()
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
	//其他类型需要自行处理
	default:
		return ""
	}
}
