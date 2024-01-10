package csv

import (
	"testing"
)

func TestCreate(t *testing.T) {

	spec := &Spec{
		FileName: "test.csv",
		Titles:   []string{"a", "b", "c", "df"},
		Data: [][]interface{}{
			{1, "1", true, "{\"k1\":\"v1\",\"k2\":2,\"k3\":{\"k31\":true},\"k4\":[1,2],\"k5\":[\"1\",\"2\"]}"},
			{2, "2", true, "{\"k1\":\"v1\",\"k2\":2,\"k3\":{\"k31\":true},\"k4\":[1,2],\"k5\":[\"1\",\"2\"]}"},
			{2.1, "2", true, "{\"k1\":\"v1\",\"k2\":2,\"k3\":{\"k31\":true},\"k4\":[1,2],\"k5\":[\"1\",\"2\"]}"},
		},
	}
	err := Create(spec)
	if err != nil {
		t.Log(err)
	}
}
