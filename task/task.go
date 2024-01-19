package task

type Task struct {
	ID         int64
	F          func(data interface{}) error
	Data       interface{}
	NeedRetry  bool
	RetryTimes int
	RetryLimit int
}
