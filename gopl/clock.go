package main

import (
	"io"
	"log"
	"net"
	"time"
)

type Clock struct{}

func (c *Clock) Start() {

	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Print(err)
			continue
		}
		go c.handle(conn) // handle connections concurrently
	}

}

func (c *Clock) handle(conn net.Conn) {
	defer conn.Close()

	for {
		_, err := io.WriteString(conn, time.Now().Format("15:04:05\n"))
		if err != nil {
			return
		}
		time.Sleep(time.Second * 1)
	}
}
