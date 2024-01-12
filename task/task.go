package monitor

type Task struct {
	ID         int64
	F          func(data interface{}) error
	Data       interface{}
	NeedRetry  bool
	RetryTimes int
	RetryLimit int
}

func NewTask(f func(data interface{}) error, data interface{}) *Task {
	return &Task{
		F:    f,
		Data: data,
	}
}
