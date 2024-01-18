package monitor

import (
	"fmt"
	"time"
)

type Worker struct {
	buckets     []chan *Task // len(buckets) eq parallelism
	parallelism int
	done        chan interface{}
}

var DefaultWorker *Worker

func init() {
	parallel := 1
	DefaultWorker = NewWorker(parallel) // todo resize
	DefaultWorker.Start()
	fmt.Printf("monitor-worker start with %d goroutines \n", parallel*2)
}

func NewWorker(parallelism int) *Worker {
	if parallelism < 1 {
		parallelism = 1024
	}
	w := &Worker{
		parallelism: parallelism,
		buckets:     make([]chan *Task, 0, parallelism),
		done:        make(chan interface{}),
	}
	for i := 0; i < parallelism; i++ {
		w.buckets = append(w.buckets, make(chan *Task))
	}
	return w
}

func (w *Worker) Start() {
	exec := func(index int) {
		defer func() {
			if fatal := recover(); fatal != nil {
				fmt.Println("monitor-worker id: ", index, " is panic! recover : ", fatal)
			}
		}()
		fmt.Println("monitor-worker id: ", index, " is start running")
		for {
			select {
			case task := <-w.buckets[index]:
				recvt := time.Now()

				fmt.Println("monitor-worker id: ", index, " is start exec task")
				err := task.F(task.Data)
				if err != nil {
					if task.NeedRetry && task.RetryTimes < task.RetryLimit {
						task.RetryTimes++
						w.buckets[index] <- task
					}
				}
				fmt.Println("monitor-worker id: ", index, " is finished task, time elapsed ns: ", time.Since(recvt).Microseconds())
			case <-w.done:
				fmt.Println("monitor-worker id: ", index, " is getting done")
				return
			}
		}
	}
	for i := 0; i < len(w.buckets); i++ {
		index := i
		go exec(index) // 1 goroutine for 1 bucket, could add more goroutine to handle 1 bucket
		go exec(index) // 2 goroutine for 1 bucket, handle for 1 goroutines timeout or hold too long
	}
}

func (w *Worker) Done() {
	for i := 0; i < (w.parallelism); i++ {
		w.done <- struct{}{}
	}
}

func (w *Worker) Put(list []*Task) {
	for _, t := range list {
		pos := t.ID % int64(len(w.buckets))
		w.buckets[pos] <- t
	}
}
