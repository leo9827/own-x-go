package timing_wheel

import (
	"fmt"
	"testing"
	"time"
)

func TestInit(t *testing.T) {
	Init()

	demoTask1 := &ScriptTask{
		user:           "dev",
		script:         "echo 'Hello octopus!'",
		timeout:        10,
		resultFilePath: "./Logs/octopus_timing_wheel.log",
	}
	AddTask(demoTask1, time.Now().Unix()+12)  // 1
	AddTask(demoTask1, time.Now().Unix()+13)  // 2
	AddTask(demoTask1, time.Now().Unix()+13)  // 3
	AddTask(demoTask1, time.Now().Unix()+45)  // 4
	AddTask(demoTask1, time.Now().Unix()+77)  // 5
	AddTask(demoTask1, time.Now().Unix()+113) // 6
	AddTask(demoTask1, time.Now().Unix()+237) // 7
	AddTask(demoTask1, time.Now().Unix()+238) // 8
	AddTask(demoTask1, time.Now().Unix()+239) // 9
	AddTask(demoTask1, time.Now().Unix()+457) // 10
	time.Sleep(3 * time.Second)
	AddTask(demoTask1, time.Now().Unix()+12)  // 11
	AddTask(demoTask1, time.Now().Unix()+457) // 12
	time.Sleep(5 * time.Second)
	AddTask(demoTask1, time.Now().Unix()+12)  // 13
	AddTask(demoTask1, time.Now().Unix()+457) // 14
	time.Sleep(7 * time.Second)
	AddTask(demoTask1, time.Now().Unix()+12)  // 15
	AddTask(demoTask1, time.Now().Unix()+457) // 16
	fmt.Println("task added, wait task execution")
	time.Sleep(100 * time.Second)
	fmt.Println("task executed, should be output 16 times")
}
