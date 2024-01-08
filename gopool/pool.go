package gopool

import (
	"context"
	"sync"
	"sync/atomic"
)

type Pool interface {
	Name() string
	Go(func())
	CtxGo(context.Context, func())
	SetPanicHandler(func(context.Context, interface{}))
	WorkerCount() int32
	SetCap(cap int32)
}

var taskPool sync.Pool

func init() {
	taskPool.New = newTask
}

type task struct {
	ctx context.Context
	f   func()

	next *task
}

func (t *task) zero() {
	t.next = nil
	t.ctx = nil
	t.f = nil
}

func (t *task) Recycle() {
	t.zero()
	taskPool.Put(t)
}

func newTask() interface{} {
	return &task{}
}

type pool struct {
	name        string
	cap         int32
	taskHead    *task
	taskTail    *task
	taskLock    sync.Mutex
	workerCount int32
	taskCount   int32
	config      *Config

	panicHandler func(context.Context, interface{})
}

func NewPool(name string, cap int32, config *Config) Pool {
	return &pool{
		name:   name,
		cap:    cap,
		config: config,
	}
}

func (p *pool) Name() string {
	return p.name
}

func (p *pool) Go(f func()) {
	p.CtxGo(context.Background(), f)
}

func (p *pool) CtxGo(ctx context.Context, f func()) {
	t := taskPool.Get().(*task)
	t.ctx = ctx
	t.f = f
	p.taskLock.Lock() // Critical Zone: update task queue
	if p.taskHead == nil {
		p.taskHead = t
		p.taskTail = t
	} else {
		p.taskTail.next = t
		p.taskTail = t
	}
	p.taskLock.Unlock()
	atomic.AddInt32(&p.taskCount, 1)

	if (atomic.LoadInt32(&p.taskCount) >= p.config.ScaleThreshold && p.WorkerCount() < atomic.LoadInt32(&p.cap)) || p.WorkerCount() == 0 {
		p.incrWorkerCount()
		w := workerPool.Get().(*worker)
		w.pool = p
		w.run()
	}
}

func (p *pool) SetPanicHandler(f func(context.Context, interface{})) {
	p.panicHandler = f
}

func (p *pool) incrWorkerCount() {
	atomic.AddInt32(&p.workerCount, 1)
}

func (p *pool) decWorkerCount() {
	atomic.AddInt32(&p.workerCount, -1)
}

func (p *pool) WorkerCount() int32 {
	return atomic.LoadInt32(&p.workerCount)
}

func (p *pool) SetCap(cap int32) {
	atomic.StoreInt32(&p.cap, cap)
}
