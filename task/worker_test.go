package monitor

import (
	"fmt"
	"testing"
	"time"
)

func TestWorker(t *testing.T) {

	w := NewWorker(4)
	w.Start()
	w.Put([]*Task{
		&Task{ID: 1, Data: "this is data", F: func(data interface{}) error {
			fmt.Println("data: ", data)
			return nil
		}}})
	w.Done()
	time.Sleep(1 * time.Second)
}
