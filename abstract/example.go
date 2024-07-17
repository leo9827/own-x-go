package main

import (
	"fmt"
	"time"
)

type Daemon interface {
	start(time.Duration)
	doWork()
}

type AbstractDaemon struct {
	Daemon
}

func (a *AbstractDaemon) start(duration time.Duration) {
	ticker := time.NewTicker(duration)

	// this will call daemon.doWork() periodically
	go func() {
		for {
			<-ticker.C
			a.doWork()
		}
	}()
}

type ConcreteDaemonA struct {
	*AbstractDaemon
	foo int
}

func newConcreteDaemonA() *ConcreteDaemonA {
	a := &AbstractDaemon{}
	r := &ConcreteDaemonA{a, 0}
	a.Daemon = r
	return r
}

type ConcreteDaemonB struct {
	*AbstractDaemon
	bar int
}

func newConcreteDaemonB() *ConcreteDaemonB {
	a := &AbstractDaemon{}
	r := &ConcreteDaemonB{a, 0}
	a.Daemon = r
	return r
}

func (a *ConcreteDaemonA) doWork() {
	a.foo++
	fmt.Println("A: ", a.foo)
}

func (b *ConcreteDaemonB) doWork() {
	b.bar--
	fmt.Println("B: ", b.bar)
}

func main() {
	var dA Daemon = newConcreteDaemonA()
	var dB Daemon = newConcreteDaemonB()

	dA.start(1 * time.Second)
	dB.start(5 * time.Second)

	time.Sleep(100 * time.Second)
}
