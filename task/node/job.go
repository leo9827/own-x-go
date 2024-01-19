package node

import (
	"fmt"
	"time"
)

type Job struct {
	IP string
}

func decodeJob(data interface{}) (*Job, error) {
	if n, ok := data.(*Job); ok {
		return n, nil
	}
	return nil, fmt.Errorf("data is invalid")
}

type Script struct {
	Script string
	Output *Output
}

func (m *Script) recv(result string) *Output {
	o := &Output{Result: result, UpdatedResultAt: time.Now()}
	m.Output = o
	return o
}

type Output struct {
	Result          string
	UpdatedResultAt time.Time
}
