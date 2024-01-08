package gopool

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestPool(t *testing.T) {
	p := NewPool("test", 100, NewConfig())
	wg := sync.WaitGroup{}
	var n int32
	for i := 0; i < 2000; i++ {
		wg.Add(1)
		p.Go(func() {
			defer wg.Done()
			atomic.AddInt32(&n, 1)
		})
	}
	wg.Wait()
	if n != 2000 {
		t.Error(n)
	}
}

func testPanicFunc() {
	panic("test")
}

func TestPoolPanic(t *testing.T) {
	p := NewPool("test", 100, NewConfig())
	p.Go(testPanicFunc)
}

const benchmarkTimes = 100000

func DoCopyStack(_, b int) int {
	if b < 100 {
		return DoCopyStack(0, b+1)
	}
	return 0
}

func testFunc() {
	DoCopyStack(0, 0)
}

func BenchmarkPool(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	config := NewConfig()
	config.ScaleThreshold = 1
	p := NewPool("benchmark", int32(runtime.GOMAXPROCS(0)), config)
	var wg sync.WaitGroup
	for i := 0; i < b.N; i++ {
		wg.Add(benchmarkTimes)
		for j := 0; j < benchmarkTimes; j++ {
			p.Go(func() {
				testFunc()
				wg.Done()
			})
		}
		wg.Wait()
	}
}

func BenchmarkPool2(b *testing.B) {
	p := NewPool("benchmark", int32(runtime.GOMAXPROCS(0)), NewConfig())

	data := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	b.ResetTimer() // 重置计时器，确保只测量任务执行的时间

	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup // WaitGroup creation inside the loop
		wg.Add(len(data))

		for _, val := range data {
			val := val
			p.Go(func(v int) func() {
				return func() {
					defer wg.Done()
					time.Sleep(time.Millisecond) // 模拟任务执行时间
				}
			}(val))
		}

		wg.Wait()
	}

	b.StopTimer() // 停止计时器
	b.ReportMetric(float64(b.N), "ns/op")
	b.ReportAllocs()
}

func BenchmarkGo(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	var wg sync.WaitGroup
	for i := 0; i < b.N; i++ {
		wg.Add(benchmarkTimes)
		for j := 0; j < benchmarkTimes; j++ {
			go func() {
				testFunc()
				wg.Done()
			}()
		}
		wg.Wait()
	}
}
