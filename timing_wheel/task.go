package timing_wheel

import "fmt"

type TimingTask interface {
	Run(timeout int)
}

type ScriptTask struct {
	user           string // default dev/mate/root
	script         string // #!/bin/bash
	timeout        int64  // default seconds
	resultFilePath string // default /var/log/octopus.log
}

func (t *ScriptTask) Run(timeout int) {
	fmt.Println("ScriptTask Run")
	fmt.Println(fmt.Sprintf("user:%s, script:%s, timeout:%d", t.user, t.script, timeout))
}
