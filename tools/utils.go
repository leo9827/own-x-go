package tools

import (
	"fmt"
	"os"
	"time"
)

func DoWithRetry(times int, interval time.Duration, f func() error) error {
	for i := 1; i <= times; i++ {
		if err := f(); err == nil {
			return nil
		} else if i == times {
			return err
		}
		time.Sleep(interval)
	}
	return nil
}

func DoWithRetryThreshold(times int, interval time.Duration, successThreshold int, f func() error) error {
	successCount := 0
	msg := ""
	for i := 1; i <= times; i++ {
		err := f()
		if err == nil {
			successCount++
			if successCount == successThreshold {
				return nil
			}
		} else {
			successCount = 0
			msg = err.Error()
		}
		time.Sleep(interval)
	}
	return fmt.Errorf("retry times exhausted, err: %s", msg)
}

// PathExist 文件是否存在
func PathExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}
