package tools

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestRenderTpl(t *testing.T) {
	tpl := "Hello, {{ replace::name }}"
	obj := map[string]interface{}{
		"name": "John",
	}
	expected := "Hello, John"

	result, err := RenderTpl(tpl, obj, false)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result != expected {
		t.Errorf("unexpected result: %s, expected: %s", result, expected)
	}

	result, err = RenderTpl("{{.HostIP}}", map[string]interface{}{"HostIP": "127.0.0.1"}, false)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != "127.0.0.1" {
		t.Errorf("unexpected result: %s, expected: %s", result, expected)
	}
	/*
		text/template 和 html/template 都是 Go 语言标准库中的模板包，用于生成文本和 HTML 输出。
		它们的主要区别在于处理输出时的安全性:
		html/template 包会自动对输出进行 HTML 转义，以防止代码注入，而 text/template 则不会进行这种转义。
	*/
	// 这里需要使用 text/template 来替代 html/template 才可以正确转义双引号 ""
	// "cluster-vip-prod-bj02": "{\"az\":\"prod_bj02\"}"
	result, err = RenderTpl(`"cluster-vip-prod-bj02":{{.JSONStr}}`, map[string]interface{}{"JSONStr": `"{\"az\":\"prod_bj02\"}"`}, false)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	expected = `"cluster-vip-prod-bj02":"{\"az\":\"prod_bj02\"}"`
	if result != expected {
		t.Errorf("unexpected result: %s, expected: %s", result, expected)
	}
}

func TestRenderTplRange(t *testing.T) {
	tpl := `{{range .Items}}{{.}},{{end}}`
	obj := map[string]interface{}{
		"Items": []string{"Item 1", "Item 2", "Item 3"},
	}
	expected := "Item 1,Item 2,Item 3,"

	result, err := RenderTpl(tpl, obj, false)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result != expected {
		t.Errorf("unexpected result: %s, expected: %s", result, expected)
	}
}

func TestRenderTplRangeStr(t *testing.T) {
	tpl := `{{range .Items}}{{.}},{{end}}`
	obj := map[string]interface{}{
		"Items": "[\"Item 1\", \"Item 2\", \"Item 3\"]",
	}

	_, err := RenderTpl(tpl, obj, false)
	if err == nil {
		t.Errorf("expected error, but got nil")
	}

}

func TestRenderTplRangeMap(t *testing.T) {
	tpl := `{{range .Items}}{{.Name}},{{.Age}},{{if .HasArrChild}}{{range .ArrChild}}{{.}}{{end}}{{end}}{{if .HasMapChild}}{{.MapChild.ChildName}}{{end}}{{end}}`
	obj := map[string]interface{}{
		"Items": []map[string]interface{}{
			{"Name": "Name1", "Age": 1},
			{"Name": "Name3", "Age": 2, "HasArrChild": true, "ArrChild": []int{1, 2, 3}},
			{"Name": "Name3", "Age": 2, "HasMapChild": true, "MapChild": map[string]interface{}{"ChildName": "childName1"}},
		},
	}

	result, err := RenderTpl(tpl, obj, false)
	t.Log(result)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != "Name1,1,Name3,2,123Name3,2,childName1" {
		t.Errorf("unexpected result: %s, expected: %s", result, "Name1,1,Name3,2,123Name3,2,childName1")
	}
}

func TestRenderTplRangeInt(t *testing.T) {
	tmpl := `{{range $index, $element := .Data}}{{$index}}{{end}}`
	obj := map[string]interface{}{
		"Data":  make([]int, 10), // 定义一个长度为10的slice
		"Data2": []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
	}

	result, err := RenderTpl(tmpl, obj, false)
	t.Log(result)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != "0123456789" {
		t.Errorf("unexpected result: %s, expected: %s", result, "0123456789")
	}
}

func TestRenderTplIfElse(t *testing.T) {
	tpl := `{{if .ShowContent}}This is the Content{{end}}`
	obj := map[string]interface{}{
		"ShowContent": true,
	}

	result, err := RenderTpl(tpl, obj, false)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result != "This is the Content" {
		t.Errorf("unexpected result: %s, expected: %s", result, "This is the Content")
	}

	// if else, Variable not null
	result, err = RenderTpl(`{{if .Name}}Hello, {{.Name}}{{else}}Hello, stranger{{end}}`, map[string]interface{}{"Name": "John"}, false)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != "Hello, John" {
		t.Errorf("unexpected result: %s, expected: %s", result, "Hello, John")
	}

	// bool true
	result, err = RenderTpl(`{{if .PrintName}}Hello, {{.Name}}{{else}}Hello, stranger{{end}}`, map[string]interface{}{"PrintName": true, "Name": "John"}, false)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != "Hello, John" {
		t.Errorf("unexpected result: %s, expected: %s", result, "Hello, John")
	}
	// bool false
	result, err = RenderTpl(`{{if .PrintName}}Hello, {{.Name}}{{else}}Hello, stranger{{end}}`, map[string]interface{}{"PrintName": false, "Name": "John"}, false)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != "Hello, stranger" {
		t.Errorf("unexpected result: %s, expected: %s", result, "Hello, stranger")
	}
	/*
		syntax: if gt arg1 arg2
		gt:	返回参数 arg1 是否大于 arg2 的布尔值。
		lt: 返回参数 arg1 是否小于 arg2 的布尔值。
		eq：返回参数 arg1 是否等于 arg2 的布尔值。
		ne：返回参数 arg1 是否不等于 arg2 的布尔值。
		le：返回参数 arg1 是否小于或等于 arg2 的布尔值。
		ge：返回参数 arg1 是否大于或等于 arg2 的布尔值。
	*/
	// gt
	result, err = RenderTpl(`{{if gt .Num 1}}Hello, {{.Name}}{{else}}Hello, stranger{{end}}`, map[string]interface{}{"Num": 10, "Name": "John"}, false)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != "Hello, John" {
		t.Errorf("unexpected result: %s, expected: %s", result, "Hello, John")
	}
	// str bool true
	result, err = RenderTpl(`{{if .StrTrue}}Hello, {{.Name}}{{else}}Hello, stranger{{end}}`, map[string]interface{}{"StrTrue": "true", "Name": "John"}, false)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != "Hello, John" {
		t.Errorf("str bool true unexpected result: %s, expected: %s", result, "Hello, John")
	}
	// str bool false -> don't support this use case
	result, err = RenderTpl(`{{if .StrTrue}}Hello, {{.Name}}{{else}}Hello, stranger{{end}}`, map[string]interface{}{"StrTrue": "false", "Name": "John"}, false)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != "Hello, John" {
		t.Errorf("str bool false unexpected result: %s, expected: %s", result, "Hello, John")
	}
}

func TestRenderTpl_Error(t *testing.T) {
	tpl := "Hello, {{ replace::name }}"
	obj := map[string]interface{}{}
	expectedErrMsg := "map has no entry for key \"name\""

	_, err := RenderTpl(tpl, obj, true)
	if err == nil {
		t.Error("expected an error, but got nil")
	} else if !strings.Contains(err.Error(), expectedErrMsg) {
		t.Errorf("unexpected error message: %s, expected: %s", err.Error(), expectedErrMsg)
	}
}

func TestParseTags(t *testing.T) {
	var metadata = map[string]string{
		"k1": "v1",
		"k2": "1",
		"k3": "true",
		"k4": "[1,2,3]",
		"k5": "[\"1\",\"2\",\"3\"]",
		"k6": "{\"k1\":\"v1\",\"k2\":\"v2\"}",
		"k7": "{\"k1\":1,\"k2\":2}",
	}

	var parsed = map[string]interface{}{}
	parsed, err := ParseTags(metadata)
	if err != nil {
		t.Errorf("unexpected err, got err: %v", err)
	}
	t.Logf("%v\n", parsed)

	if parsed["k1"] != "v1" {
		t.Errorf("unexpected result: %s, expected: %s", parsed["k1"], "v1")
	}
	if parsed["k2"] != 1 {
		t.Errorf("unexpected result: %s, expected: %d", parsed["k2"], 1)
	}
	if parsed["k3"] != true {
		t.Errorf("unexpected result: %s, expected: %v", parsed["k3"], true)
	}

	k4typeOf := reflect.TypeOf(parsed["k4"])
	if k4typeOf.Kind() != reflect.Slice && k4typeOf.Kind() != reflect.Array {
		t.Errorf("unexpected result: %s, expected: %s", k4typeOf.Kind(), reflect.Slice)
	}
	//var arr2 = []string{"1", "2", "3"}
	k5typeOf := reflect.TypeOf(parsed["k5"])
	if k5typeOf.Kind() != reflect.Slice {
		t.Errorf("unexpected result: %s, expected: %s", k5typeOf.Kind(), reflect.Slice)
	}
}

func TestParseValueDemo(t *testing.T) {
	var metadata = map[string]string{
		"k1": "v1",
		"k2": "1",
		"k3": "true",
		"k4": "[1,2,3]",
		"k5": "[\"1\",\"2\",\"3\"]",
		"k6": "{\"k1\":\"v1\",\"k2\":\"v2\"}",
		"k7": "{\"k1\":1,\"k2\":2}",
	}

	var parsed = map[string]interface{}{}
	for k, v := range metadata {
		parsed[k] = parseValueDemo(v)
	}

	t.Logf("%v\n", parsed)

	if parsed["k1"] != "v1" {
		t.Errorf("unexpected result: %s, expected: %s", parsed["k1"], "v1")
	}
	if parsed["k2"] != 1 {
		t.Errorf("unexpected result: %s, expected: %d", parsed["k2"], 1)
	}
	if parsed["k3"] != true {
		t.Errorf("unexpected result: %s, expected: %v", parsed["k3"], true)
	}

	k4typeOf := reflect.TypeOf(parsed["k4"])
	if k4typeOf.Kind() != reflect.Slice && k4typeOf.Kind() != reflect.Array {
		t.Errorf("unexpected result: %s, expected: %s", k4typeOf.Kind(), reflect.Slice)
	}
	//var arr2 = []string{"1", "2", "3"}
	k5typeOf := reflect.TypeOf(parsed["k5"])
	if k5typeOf.Kind() != reflect.Slice {
		t.Errorf("unexpected result: %s, expected: %s", k5typeOf.Kind(), reflect.Slice)
	}
}

func TestFuncs(t *testing.T) {
	tpl := `"dp": [{{ (Int64 .core_num_dv) }}]` // pass
	result, err := RenderTpl(tpl, map[string]interface{}{"core_num_dv": 10}, false)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != `"dp": [10]` {
		t.Errorf("unexpected result: %s, expected: %s", result, `"dp": [10]`)
	} else {
		t.Log("tpl:", tpl, "\tresult", result)
	}
	tpl = `"dp": [{{ Plus (Int64 .core_num_dv) 1 }}]` // pass
	result, err = RenderTpl(tpl, map[string]interface{}{"core_num_dv": 10}, false)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != `"dp": [11]` {
		t.Errorf("unexpected result: %s, expected: %s", result, `"dp": [11]`)
	} else {
		t.Log("tpl:", tpl, "\tresult", result)
	}
	tpl = `"dp": [{{ (Plus (Int64 .core_num_dv) 1) }}]` // pass
	result, err = RenderTpl(tpl, map[string]interface{}{"core_num_dv": 10}, false)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != `"dp": [11]` {
		t.Errorf("unexpected result: %s, expected: %s", result, `"dp": [11]`)
	} else {
		t.Log("tpl:", tpl, "\tresult", result)
	}
	tpl = `"dp": [{{ (Iterate 1 (Plus (Int64 .core_num_dv) 1)) }}]` // pass "dp": [[1 2 3 4 5 6 7 8 9 10]]
	result, err = RenderTpl(tpl, map[string]interface{}{"core_num_dv": 10}, false)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != `"dp": [[1 2 3 4 5 6 7 8 9 10]]` {
		t.Errorf("unexpected result: %s, expected: %s", result, `"dp": [[1 2 3 4 5 6 7 8 9 10]]`)
	} else {
		t.Log("tpl:", tpl, "\tresult", result)
	}
	tpl = `"dp": [{{ range $1 := (Iterate 1 (Plus (Int64 .core_num_dv) 1)) }} {{.}} {{end}} ]` // pass "dp": [ 1  2  3  4  5  6  7  8  9  10  ]
	result, err = RenderTpl(tpl, map[string]interface{}{"core_num_dv": 10}, false)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != `"dp": [ 1  2  3  4  5  6  7  8  9  10  ]` {
		t.Errorf("unexpected result: %s, expected: %s", result, `"dp": [ 1  2  3  4  5  6  7  8  9  10  ]`)
	} else {
		t.Log("tpl:", tpl, "\tresult", result)
	}
	tpl = `"dp": [{{ range $i,$v :=(Iterate 1 (Plus (Int64 .core_num_dv) 1)) }}  i:{{$i}} v:{{$v}} {{end}}  ]` // pass "dp": [  i:0 v:1   i:1 v:2   i:2 v:3   i:3 v:4   i:4 v:5   i:5 v:6   i:6 v:7   i:7 v:8   i:8 v:9   i:9 v:10   ]
	result, err = RenderTpl(tpl, map[string]interface{}{"core_num_dv": 10}, false)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != `"dp": [  i:0 v:1   i:1 v:2   i:2 v:3   i:3 v:4   i:4 v:5   i:5 v:6   i:6 v:7   i:7 v:8   i:8 v:9   i:9 v:10   ]` {
		t.Errorf("unexpected result: %s, expected: %s", result, `"dp": [  i:0 v:1   i:1 v:2   i:2 v:3   i:3 v:4   i:4 v:5   i:5 v:6   i:6 v:7   i:7 v:8   i:8 v:9   i:9 v:10   ]`)
	} else {
		t.Log("tpl:", tpl, "\tresult", result)
	}
}

func TestMustKey(t *testing.T) {
	result, err := RenderTpl("{{.HostIP}} {{.missedKey}}", map[string]interface{}{"HostIP": "127.0.0.1"}, true)
	// assert
	if err != nil {
		replaced := strings.Replace(err.Error(), "map has no entry for key", "not found input of key", -1)
		t.Logf("%v", replaced)
	} else {
		t.Errorf("unexpected error: %v", err)
	}
	t.Logf(result)

}

// BenchmarkParseValue-12    	  130375	      7816 ns/op

func BenchmarkParseValueDemo(b *testing.B) {
	var metadata = map[string]string{
		"k1": "v1",
		"k2": "1",
		"k3": "true",
		"k4": "[1,2,3]",
		"k5": "[\"1\",\"2\",\"3\"]",
		"k6": "{\"k1\":\"v1\",\"k2\":\"v2\"}",
		"k7": "{\"k1\":1,\"k2\":2}",
	}

	for i := 0; i < b.N; i++ {
		var parsed = map[string]interface{}{}
		for k, v := range metadata {
			parsed[k] = parseValueDemo(v)
		}
	}
}

// BenchmarkParseValue2-12    	  211285	      5276 ns/op
func BenchmarkParseValueDemo2(b *testing.B) {
	var metadata = map[string]string{
		"k1": "v1",
		"k2": "1",
		"k3": "true",
		"k4": "[1,2,3]",
		"k5": "[\"1\",\"2\",\"3\"]",
		"k6": "{\"k1\":\"v1\",\"k2\":\"v2\"}",
		"k7": "{\"k1\":1,\"k2\":2}",
	}

	for i := 0; i < b.N; i++ {
		var parsed = map[string]interface{}{}
		for k, v := range metadata {
			parsed[k] = parseValueDemo2(v)
		}
	}
}

// BenchmarkParseTags-12    	  235759	      4795 ns/op
func BenchmarkParseTags(b *testing.B) {
	var tags = map[string]string{
		"k1": "v1",
		"k2": "1",
		"k3": "true",
		"k4": "[1,2,3]",
		"k5": "[\"1\",\"2\",\"3\"]",
		"k6": "{\"k1\":\"v1\",\"k2\":\"v2\"}",
		"k7": "{\"k1\":1,\"k2\":2}",
	}
	for i := 0; i < b.N; i++ {
		_, _ = ParseTags(tags)
	}
}

func parseValueDemo(v string) interface{} {
	// First try to convert to integer
	if i, err := strconv.Atoi(v); err == nil {
		return i
	}

	// Then try to convert to boolean
	if b, err := strconv.ParseBool(v); err == nil {
		return b
	}

	// Then try to convert to array
	var arr []interface{}
	if err := json.Unmarshal([]byte(v), &arr); err == nil {
		for i, elem := range arr {
			if str, ok := elem.(string); ok {
				if num, err := strconv.Atoi(str); err == nil {
					arr[i] = num
				}
			}
		}
		return arr
	}

	// Then try to convert to dictionary
	var dict map[string]interface{}
	if err := json.Unmarshal([]byte(v), &dict); err == nil {
		for key, val := range dict {
			if str, ok := val.(string); ok {
				if num, err := strconv.Atoi(str); err == nil {
					dict[key] = num
				}
			}
		}
		return dict
	}

	// If all conversions failed, return string
	return v
}

func parseValueDemo2(v string) interface{} {
	// First try to convert to integer
	if i, err := strconv.Atoi(v); err == nil {
		return i
	}

	// Then try to convert to boolean
	if b, err := strconv.ParseBool(v); err == nil {
		return b
	}

	// Then try to convert to array or dictionary
	var arrOrDict interface{}
	if err := json.Unmarshal([]byte(v), &arrOrDict); err == nil {
		switch arrOrDict := arrOrDict.(type) {
		case []interface{}:
			for i, elem := range arrOrDict {
				if str, ok := elem.(string); ok {
					if num, err := strconv.Atoi(str); err == nil {
						arrOrDict[i] = num
					}
				}
			}
			return arrOrDict
		case map[string]interface{}:
			for key, val := range arrOrDict {
				if str, ok := val.(string); ok {
					if num, err := strconv.Atoi(str); err == nil {
						arrOrDict[key] = num
					}
				}
			}
			return arrOrDict
		}
	}

	// If all conversions failed, return string
	return v
}
