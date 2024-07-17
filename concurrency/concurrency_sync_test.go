package concurrency

import (
	"fmt"
	"log"
	"math"
	"net"
	"os"
	"runtime"
	"sync"
	"testing"
	"text/tabwriter"
	"time"
)

func TestWaitGroup(t *testing.T) {
	sayHello := func(wg *sync.WaitGroup) {
		defer wg.Done()
		fmt.Println("hello")
	}
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go sayHello(wg)
	wg.Wait()
}

func TestWaitGroup2(t *testing.T) {
	type Example struct {
		Name string
		IntF int
		ArrF []string
	}
	var (
		wg       sync.WaitGroup
		example  Example
		example2 *Example
		intv     int
		str      string
		mu       sync.Mutex
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		mu.Lock()
		fmt.Println("goroutine 1 locked")
		example = Example{
			Name: "this name is assign in goroutine",
		}
		mu.Unlock()
		example2 = &Example{
			Name: "this ptr is assign in goroutine",
		}
		intv = 111
		str = "str111"
	}()
	go func(example3 *Example) {
		defer wg.Done()
		//time.Sleep(time.Millisecond)
		mu.Lock()
		fmt.Println("goroutine 2 locked")
		example3.Name = "this is pass value in param"
		mu.Unlock()
	}(&example)
	wg.Wait()
	fmt.Printf("after wg, example is: %v\n", example.Name)
	fmt.Printf("after wg, example2 is: %v\n", example2)
	fmt.Printf("after wg, intv is: %v\n", intv)
	fmt.Printf("after wg, str is: %v\n", str)
}

func TestClose(t *testing.T) {
	wg := &sync.WaitGroup{}
	for _, salutation := range []string{"hello", "greetings", "good day"} {
		wg.Add(1)
		go func(s string) {
			defer wg.Done()
			fmt.Println(s) // not in closure
		}(salutation)
	}
	wg.Wait()

	var wg2 sync.WaitGroup
	for _, salutation := range []string{"1-hello", "1-greetings", "1-good day"} {
		wg2.Add(1)
		go func(salutation string) {
			defer wg2.Done()
			fmt.Println(salutation)
		}(salutation)
	}
	wg2.Wait()
}

func TestMem(t *testing.T) {

	memConsumed := func() uint64 {
		runtime.GC()
		var s runtime.MemStats
		runtime.ReadMemStats(&s)
		return s.Sys
	}
	var c <-chan interface{}
	var wg sync.WaitGroup
	noop := func() { wg.Done(); <-c }
	const numGoroutines = 1e4
	wg.Add(numGoroutines)
	before := memConsumed()
	for i := numGoroutines; i > 0; i-- {
		go noop()
	}
	wg.Wait()
	after := memConsumed()
	fmt.Printf("%.3fkb", float64(after-before)/numGoroutines/1000)
}

func BenchmarkContextSwitch(b *testing.B) {
	var wg sync.WaitGroup
	begin := make(chan struct{})
	c := make(chan struct{})
	var token struct{}
	sender := func() {
		defer wg.Done()
		<-begin
		for i := 0; i < b.N; i++ {
			c <- token
		}
	}
	receiver := func() {
		defer wg.Done()
		<-begin
		for i := 0; i < b.N; i++ {
			<-c
		}
	}

	wg.Add(2)
	go sender()
	go receiver()
	b.StartTimer()
	close(begin) // close begin chan to start sender & receiver
	wg.Wait()
}

func TestMutex(t *testing.T) {
	var count int
	var l sync.Mutex

	increment := func() {
		defer l.Unlock()
		l.Lock()
		count++
		fmt.Printf("Incrementing: %d\n", count)
	}

	decrement := func() {
		defer l.Unlock()
		l.Lock()
		count--
		fmt.Printf("Decrementing: %d\n", count)
	}

	var arithmetic sync.WaitGroup
	for i := 0; i < 5; i++ {
		arithmetic.Add(1)
		go func() {
			defer arithmetic.Done()
			increment()
		}()
	}

	for i := 0; i < 5; i++ {
		arithmetic.Add(1)
		go func() {
			defer arithmetic.Done()
			decrement()
		}()
	}

	arithmetic.Wait()
	fmt.Println("Arithmetic complete. count: ", count)
}

func TestMutex2(t *testing.T) {
	producer := func(wg *sync.WaitGroup, l sync.Locker) {
		defer wg.Done()
		for i := 0; i < 5; i++ {
			l.Lock()
			defer l.Unlock()
			time.Sleep(1 * time.Second)
		}
	}

	observer := func(wg *sync.WaitGroup, l sync.Locker) {
		defer wg.Done()
		l.Lock()
		defer l.Unlock()
	}

	test := func(count int, mutex, rwMutex sync.Locker) time.Duration {
		var wg sync.WaitGroup
		wg.Add(count + 1)
		beginTestTime := time.Now()
		go producer(&wg, mutex)
		for i := count; i > 0; i-- {
			go observer(&wg, rwMutex)

		}
		wg.Wait()
		return time.Since(beginTestTime)
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 1, 2, ' ', 0)
	defer tw.Flush()

	var m sync.RWMutex
	fmt.Fprintf(tw, "Readers\tRWMutext\tMutex\n")
	for i := 0; i < 20; i++ {
		count := int(math.Pow(2, float64(i)))
		fmt.Fprintf(
			tw,
			"%d\t%v\t%v\n",
			count,
			test(count, &m, m.RLocker()),
			test(count, &m, &m),
		)
	}
}

func TestCond(t *testing.T) {
	conditionTrue := func() bool {
		return true
	}

	c := sync.NewCond(&sync.Mutex{})
	c.L.Lock()
	for conditionTrue() == false {
		c.Wait()
	}
	c.L.Unlock()
}

func TestCond2(t *testing.T) {
	c := sync.NewCond(&sync.Mutex{})

	queue := make([]interface{}, 0, 10)

	removeFromQueue := func(delay time.Duration) {
		time.Sleep(delay)
		c.L.Lock()
		queue = queue[1:]
		fmt.Println("Removed from queue")
		c.L.Unlock()
		c.Signal()
	}

	for i := 0; i < 10; i++ {
		c.L.Lock()
		for len(queue) == 2 {
			c.Wait()
		}
		fmt.Println("Adding to queue")
		queue = append(queue, struct{}{})
		go removeFromQueue(1 * time.Second)
		c.L.Unlock()
	}
}

func TestCond3(t *testing.T) {
	type Button struct {
		Clicked *sync.Cond
	}
	button := Button{Clicked: sync.NewCond(&sync.Mutex{})}

	subscribe := func(c *sync.Cond, fn func()) {
		var goroutineRunning sync.WaitGroup
		goroutineRunning.Add(1)
		go func() {
			goroutineRunning.Done()
			c.L.Lock()
			defer c.L.Unlock()
			c.Wait()
			fn()
		}()
		goroutineRunning.Wait()
	}

	var clickRegistered sync.WaitGroup
	clickRegistered.Add(3)
	subscribe(button.Clicked, func() {
		fmt.Println("Maximizing window.")
		clickRegistered.Done()
	})
	subscribe(button.Clicked, func() {
		fmt.Println("Displaying annoying dialog box!")
		clickRegistered.Done()
	})
	subscribe(button.Clicked, func() {
		fmt.Println("Mouse clicked")
	})

	button.Clicked.Broadcast()
	clickRegistered.Wait()
}

func TestOnce(t *testing.T) {
	var count int
	increment := func() {
		count++
	}

	var once sync.Once

	var increments sync.WaitGroup
	increments.Add(100)
	for i := 0; i < 100; i++ {
		go func() {
			defer increments.Done()
			once.Do(increment)
		}()
	}

	increments.Wait()
	fmt.Printf("Count is %d\n", count) // Count is 1
}

func TestOnce2(t *testing.T) {
	var count int
	increment := func() { count++ }
	decrement := func() { count-- }

	var once sync.Once
	once.Do(increment)
	once.Do(decrement)

	fmt.Printf("Count is %d\n", count) // Count is 1
}

func TestPool(t *testing.T) {
	myPool := &sync.Pool{
		New: func() interface{} {
			fmt.Println("Creating new instance")
			return struct{}{}
		},
	}

	myPool.Get()             // Creating new instance
	instance := myPool.Get() // Creating new instance
	myPool.Put(instance)
	myPool.Get() // nothing out put
}

func TestPool2(t *testing.T) {
	var numCalcsCreated int
	calcPool := &sync.Pool{
		New: func() interface{} {
			numCalcsCreated += 1
			mem := make([]byte, 1024) // 1kb
			return &mem
		},
	}

	// Seed the pool with 4kb
	calcPool.Put(calcPool.New())
	calcPool.Put(calcPool.New())
	calcPool.Put(calcPool.New())
	calcPool.Put(calcPool.New())

	const numWorkers = 1024 * 1024
	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for i := numWorkers; i > 0; i-- {
		go func() {
			defer wg.Done()
			mem := calcPool.Get().(*[]byte)
			defer calcPool.Put(mem)
			// Assume something interesting, but quick is being done with this memory.
		}()
	}

	wg.Wait()
	fmt.Printf("%d calculators were created.\n", numCalcsCreated) // 16
}

func TestPool3(t *testing.T) {
	warmServiceConnCache := func() *sync.Pool {
		p := &sync.Pool{
			New: func() interface{} {
				time.Sleep(1 * time.Second)
				return struct{}{}
			},
		}
		for i := 0; i < 10; i++ {
			p.Put(p.New())
		}
		return p
	}
	startNetworkDaemon := func() *sync.WaitGroup {
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			connPool := warmServiceConnCache()
			server, err := net.Listen("tcp", "localhost:8080")
			if err != nil {
				log.Fatal(err)
			}
			defer server.Close()
			wg.Done()

			for {
				conn, err := server.Accept()
				if err != nil {
					log.Printf("cannot accept connection: %v", err)
					continue
				}
				svcConn := connPool.Get()
				fmt.Fprintln(conn, "")
				connPool.Put(svcConn)
				conn.Close()
			}
		}()
		return &wg
	}
	startNetworkDaemon()
}
