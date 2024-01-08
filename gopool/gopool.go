package gopool

import "context"

var defaultPool Pool

func init() {
	defaultPool = NewPool("default", 1000, NewConfig())
}

func Go(f func()) {
	CtxGo(context.Background(), f)
}

func CtxGo(ctx context.Context, f func()) {
	defaultPool.CtxGo(ctx, f)
}

// SetCap is not recommended to be called, this func changes the global pool's capacity which will affect other callers.
func SetCap(cap int32) {
	defaultPool.SetCap(cap)
}

// SetPanicHandler sets the panic handler for the global pool.
func SetPanicHandler(f func(context.Context, interface{})) {
	defaultPool.SetPanicHandler(f)
}

// WorkerCount returns the number of global default pool's running workers
func WorkerCount() int32 {
	return defaultPool.WorkerCount()
}
