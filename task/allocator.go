package task

type Allocator interface {
	StartAlloc(round int64)
	RegistryTaskBuild(taskBuild func(round int64, data interface{}) ([]*Task, error)) error
	RegistryCutInLineTaskBuild(taskBuild func(data interface{}) ([]*Task, error)) error
	RegistryWorker(worker *Worker) error
}
