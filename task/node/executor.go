package node

import (
	"fmt"
	"ownx/ssh"
	"ownx/task"
	"time"
)

type Executor struct {
	Name string
}

var DefaultNodeExecutor = &Executor{Name: "default-node-executor"}

func (e *Executor) Exec(jobData interface{}) error {
	now := time.Now()

	job, err := decodeJob(jobData)
	if err != nil {
		return err
	}

	defer func(ip string) {
		fmt.Println("Job Exec, ip: ", ip, " time elapsed ms: ", time.Since(now).Milliseconds())
	}(job.IP)

	scripts := make([]*Script, 0)
	s := &Script{
		Script: "echo 'hello!'",
		Output: nil,
	}

	scripts = append(scripts, s)
	outputs := e.remoteExec(job.IP, scripts)
	err = e.updateResult(job.IP, outputs)
	if err != nil {
		return err
	}
	return nil
}

func (e *Executor) remoteExec(ip string, scripts []*Script) []*Output {
	now := time.Now()
	defer func() {
		fmt.Println("remoteExec in ip: ", ip, " time elapsed ms: ", time.Since(now).Milliseconds())
	}()

	outputs := make([]*Output, 0, len(scripts))

	cmd := ssh.NewCmd(fmt.Sprintf("%s:%s", ip, "port")).
		KeyFile(""). // should not be empty
		ConnectTimeout(5 * time.Second).
		IoTimeout(5 * time.Second)
	if _, err := cmd.Connect(); err != nil {
		msg := fmt.Sprintf("connect to host %s failed. err=%v", ip, err)
		for _, script := range scripts {
			outputs = append(outputs, script.recv(msg))
		}
		return outputs
	}

	defer cmd.Close()
	for _, script := range scripts {
		ret := cmd.RunCmd(fmt.Sprintf(script.Script))
		ret.HasError(true)
		outputs = append(outputs, script.recv(ret.GetStdout()))
	}
	return outputs
}

func (e *Executor) updateResult(ip string, outputs []*Output) error {
	if len(outputs) == 0 {
		return nil
	}
	for _, o := range outputs {
		fmt.Printf("%s exec got ouput: %s", ip, o.Result)
	}
	return nil
}

// TaskBuild should register in job_retriever for build task
func (e *Executor) TaskBuild(round int64, data interface{}) ([]*task.Task, error) {
	fmt.Println("Executor TaskBuild, round: ", round)

	ipList, ok := data.([]string)
	if !ok {
		return nil, fmt.Errorf("data is not []string")
	}

	tasks := make([]*task.Task, 0, len(ipList))
	for _, ip := range ipList {
		tasks = append(tasks, &task.Task{
			ID:         time.Now().Unix(),
			F:          e.Exec,
			Data:       &Job{IP: ip},
			NeedRetry:  true,
			RetryTimes: 0,
			RetryLimit: 3,
		})
	}

	return tasks, nil
}

// CutInLineTaskBuild 插队任务
func (e *Executor) CutInLineTaskBuild(nodeIPList interface{}) ([]*task.Task, error) {
	return e.TaskBuild(-1, nodeIPList)
}
