package gopool

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type pool struct {
	queue chan []byte
}

var defaultPool = &pool{}

func Go(f func()) {

}

func (p *pool) run() {
	for {
		select {
		case <-time.After(1 * time.Second):
			fmt.Println("after 1 second")
		case job := <-p.queue:
			fmt.Println(job)
			signalChan := make(chan os.Signal, 1)
			signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
		}
	}
}

type Pool interface {
	Submit(f func())
	Go(f func())
	CtxGo(ctx context.Context, f func())
}
