package concurrency

import (
	"bytes"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"
)

func TestChan1(t *testing.T) {
	stringStream := make(chan string)
	go func() {
		stringStream <- "Hello channels!"
	}()
	fmt.Println(<-stringStream) // blocking util receive message from channel
}

func TestChan2(t *testing.T) {
	writeStream := make(chan<- interface{})
	readStream := make(<-chan interface{})

	//<-writeStream            //invalid operation: <-writeStream (receive from send-only type  chan<- interface {})
	//readStream <- struct{}{} //invalid operation: readStream <- struct {} literal (send to receive-only type <-chan interface {})

	writeStream <- struct{}{}
	<-readStream
}

func TestChan3Deadlock(t *testing.T) {
	// deadlock
	stringStream := make(chan string)
	go func() {
		if 0 != 1 {
			return
		}
		stringStream <- "Hello channels!"
	}()
	fmt.Println(<-stringStream) // always blocking
}

func TestChan4(t *testing.T) {
	stringStream := make(chan string)
	go func() {
		stringStream <- "Hello channels!"
	}()
	salutation, ok := <-stringStream
	fmt.Printf("(%v): %v\n", ok, salutation) // output: (true): Hello channels!

	intStream := make(chan int)
	close(intStream)
	integer, ok := <-intStream
	fmt.Printf("(%v): %v\n", ok, integer) // output: (false): 0
}

func TestChan5(t *testing.T) {
	intStream := make(chan int)
	go func() {
		defer close(intStream) // if not close, main foreach will be blocking
		for i := 0; i < 10; i++ {
			intStream <- i
		}
	}()

	for integer := range intStream {
		fmt.Printf("%v ", integer)
	}
	fmt.Println("\nafter foreach")
}

func TestChan6(t *testing.T) {
	begin := make(chan interface{})
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-begin // blocking until close(begin)
			fmt.Printf("%v has begun\n", i)
		}(i)
	}

	fmt.Println("Unblocking goroutines...")
	close(begin)
	wg.Wait()
}

func TestChan7(t *testing.T) {
	var stdoutBuff bytes.Buffer
	defer stdoutBuff.WriteTo(os.Stdout)

	intStream := make(chan int, 4)
	go func() {
		defer close(intStream)
		defer fmt.Fprintln(&stdoutBuff, "Producer Done.")
		for i := 0; i < 5; i++ {
			fmt.Fprintf(&stdoutBuff, "Sending: %d\n", i)
			intStream <- i
		}
	}()

	for integer := range intStream {
		fmt.Fprintf(&stdoutBuff, "Received %v.\n", integer)
	}
}

func TestChan8(t *testing.T) {
	chanOwner := func() <-chan int {
		resultStream := make(chan int, 5)
		go func() {
			defer close(resultStream)
			for i := 0; i <= 5; i++ {
				resultStream <- i
			}
		}()
		return resultStream
	}
	resultStream := chanOwner()
	for integer := range resultStream {
		fmt.Printf("Received: %d\n", integer)
	}
	fmt.Println("Data received.")
}

func TestSelect1(t *testing.T) {
	start := time.Now()
	c := make(chan interface{})
	go func() {
		time.Sleep(3 * time.Second)
		close(c)
	}()

	fmt.Println("Blocking on read...")
	<-c
	fmt.Printf("Unblocked %v later.\n", time.Since(start))
}

func TestSelect2(t *testing.T) {
	c1 := make(chan int)
	close(c1)
	c2 := make(chan int)
	close(c2)

	var count1, count2 int
	for i := 1000; i >= 0; i-- {
		select {
		case <-c1:
			count1++
		case <-c2:
			count2++
		}
		//select 语句会从上到下依次检查每个 case 分支：
		//	如果通道已准备好发送或接收数据，且没有阻塞，则对应的 case 分支会被选中。
		//	如果有多个 case 分支同时满足条件，那么将随机选择一个分支执行。
		//一旦选择了某个分支并执行其中的代码块，select 语句就会立即结束，并不会继续检查其他分支。

		//由于通道 c1 和 c2 在循环开始前都已经被关闭，因此在每次循环迭代中，这两个通道都会返回已关闭的状态。
		//因此，无论是 c1 还是 c2 分支都会被执行，而且它们的执行顺序是随机的。
	}
	fmt.Printf("c1Count: %d\nc2Count: %d\n", count1, count2)
}

func TestSelect3(t *testing.T) {
	done := make(chan interface{})
	go func() {
		time.Sleep(5 * time.Second)
		close(done)
	}()
	workCounter := 0

loop:
	for {
		select {
		case <-done:
			break loop
		default:
		}
		workCounter++
		time.Sleep(1 * time.Second)
	}

	fmt.Printf("Achieved %v cycles of work before signalled to stop.\n", workCounter)
}
