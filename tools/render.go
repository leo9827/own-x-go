package tools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	. "text/template"
)

var varPattern = regexp.MustCompile(`{{\s*replace::(\w+)\s*}}`)

func RenderTpl(tpl string, obj map[string]interface{}, mustKey bool) (string, error) {
	var err error
	realTpl := varPattern.ReplaceAllStringFunc(tpl, func(field string) string {
		substr, subErr := renderExt(strings.Replace(field, "replace::", ".", 1), obj, mustKey)
		if subErr != nil {
			err = subErr
		}
		return substr
	})
	if err != nil {
		return "", err
	}
	return renderExt(realTpl, obj, mustKey)
}

func renderExt(data string, obj map[string]interface{}, mustKey bool) (string, error) {
	opt := "missingkey=default"
	if mustKey {
		opt = "missingkey=error"
	}
	template, err := New("").Option(opt).Funcs(GetFuncs()).Parse(data)
	if err != nil {
		return "", fmt.Errorf("parse failed. %w", err)
	}
	buffer := bytes.NewBufferString("")
	if err = template.Execute(buffer, obj); err != nil {
		return "", fmt.Errorf("render failed in %w", err)
	}
	return buffer.String(), nil
}

const (
	SuffixObj = ":obj"
	//PackageRpm = "rpm"
	//SuffixTpl  = ".tpl"
	//BoolTrue   = "true"
	//BoolFalse  = "false"
	//PrefixArr  = "["
	//SuffixArr  = "]"
	//PrefixMap  = "{"
	//SuffixMap  = "}"
	//REGEXP     = "([1-9]|true|false|\\[\\]|{})"
)

func ParseTags(tags map[string]string) (map[string]interface{}, error) {
	parsed := make(map[string]interface{}, len(tags))

	for k, v := range tags {
		if strings.HasSuffix(k, SuffixObj) {
			subObj := make(map[string]interface{})
			if err := json.Unmarshal([]byte(v), &subObj); err != nil {
				return nil, fmt.Errorf("[prepare] failed to unmarshal tag. key=%s. %w", k, err)
			}
			parsed[strings.TrimSuffix(k, SuffixObj)] = subObj
		} else {
			parsed[k] = parseValue(v)
		}
	}

	return parsed, nil
}

func parseValue(v string) interface{} {
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
		return arrOrDict
		//switch arrOrDict := arrOrDict.(type) {
		//case []interface{}:
		//	for i, elem := range arrOrDict {
		//		if str, ok := elem.(string); ok {
		//			if num, err := strconv.Atoi(str); err == nil {
		//				arrOrDict[i] = num
		//			}
		//			if b, err := strconv.ParseBool(str); err == nil {
		//				arrOrDict[i] = b
		//			}
		//		}
		//	}
		//	return arrOrDict
		//case map[string]interface{}:
		//	for key, val := range arrOrDict {
		//		if str, ok := val.(string); ok {
		//			if num, err := strconv.Atoi(str); err == nil {
		//				arrOrDict[key] = num
		//			}
		//			if b, err := strconv.ParseBool(str); err == nil {
		//				arrOrDict[key] = b
		//			}
		//		}
		//	}
		//	return arrOrDict
		//}
	}

	// If all conversions failed, return string
	return v
}
