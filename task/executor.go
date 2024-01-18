package monitor

type Executor interface {
	Exec(data interface{}) error
	TaskBuild(round int64, input interface{}) ([]*Task, error)
	CutInLineTaskBuild(input interface{}) ([]*Task, error)
}
