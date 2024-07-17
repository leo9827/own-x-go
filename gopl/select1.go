package main

import (
	"fmt"
	"os"
	"time"
)

type Select struct{}

func (s *Select) Start() {
	ch := make(chan int, 1)
	for i := 0; i < 10; i++ {
		select {
		case x := <-ch:
			fmt.Print("x=", x) // 0 2 4 6 8
			fmt.Print(" i=", i, "\n")
		case ch <- i:
		}
	}
}

func (s *Select) runLaunch() {
	fmt.Println("Commencing countdown.")
	tick := time.Tick(1 * time.Second)
	for countdown := 10; countdown > 0; countdown-- {
		fmt.Println(countdown)
		<-tick
	}

	fmt.Println("Launch!")

	abort := make(chan struct{})
	go func() {
		_, _ = os.Stdin.Read(make([]byte, 1))
		abort <- struct{}{}
	}()

	select {
	case <-time.After(10 * time.Second):
		fmt.Println("Launch!")
	case <-abort:
		fmt.Println("Launch aborted!")
		return
	}
	s.launch()
}

func (s *Select) launch() {
	fmt.Println("Lift off!")
}
