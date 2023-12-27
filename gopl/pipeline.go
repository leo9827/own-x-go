package main

import "fmt"

func main() {
	natural := make(chan int)
	square := make(chan int)

	go func() {
		for x := 0; ; x++ {

			natural <- x
		}

	}()

	go func() {
		for {
			msg := <-natural
			square <- msg
		}
	}()

	// in main goroutine
	msg := <-square
	fmt.Println(msg)
}
