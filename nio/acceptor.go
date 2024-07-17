package nio

import (
	"bytes"
	"encoding/binary"
	"errors"
	"log"
	"net"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type socket struct {
	addr    string
	handler *Handler
	closed  bool
	ordered bool // true: handler msg in serial，false: handle msg in concurrency
}

type SocketAcceptor struct {
	socket
	lock     sync.Mutex
	sessions map[int64]*Session // save all sessions
	acceptor net.Listener
}

const tcp string = "tcp"

func CreateAcceptor(ip, port string, handler *Handler, codec Codec, ordered bool) (*SocketAcceptor, error) {
	if handler == nil {
		return nil, errors.New(ip + ":" + port + " Create socket acceptor failed. handler is nil.")
	}
	acceptor := &SocketAcceptor{}
	acceptor.addr = ip + ":" + port
	acceptor.handler = handler
	acceptor.ordered = ordered
	acceptor.sessions = make(map[int64]*Session)
	acceptor.handler.codec = codec
	if codec == nil {
		acceptor.handler.codec = getDefaultCodec(false)
	}
	return acceptor, nil
}

func (socket *SocketAcceptor) Open() error {

	log.Printf("Open socket %s acceptor and listening...", socket.addr)

	acceptor, err := net.Listen(tcp, socket.addr)
	if err != nil {
		log.Printf("Open socket %s acceptor failed. %s", socket.addr, err.Error())
		return err
	}
	socket.acceptor = acceptor

	log.Printf("Open socket %s acceptor success.", socket.addr)

	go func() {
		defer OnError("")
		socket.handlerConnection()
	}()

	return nil
}

func (socket *SocketAcceptor) handlerConnection() {
	var sessionId int64 = 0

	for {

		if socket.closed {
			log.Printf("[server] Socket acceptor %s is closed. Normal finished.", socket.addr)
			return
		}
		connection, err := socket.acceptor.Accept()
		if err != nil {
			log.Printf("[server] [%s] Open socket connection failed. %s", socket.addr, err.Error())
			break
		}

		session := &Session{}
		session.id = atomic.AddInt64(&sessionId, 1)
		session.isClient = false
		session.codec = socket.handler.codec
		session.acceptor = socket
		session.connection = connection

		if sendErr := socket.sendSessionId(connection, session.id); sendErr != nil {
			session.id = atomic.AddInt64(&sessionId, -1)
			err := connection.Close()
			if err != nil {
				log.Printf("[server] [%s] Close socket connection error. %s", socket.addr, err.Error())
			}
			continue
		}
		log.Printf("[server] [%s] [%d] Accept connection from remote %s",
			socket.addr, session.id, connection.RemoteAddr())
		socket.addSession(session)
		if socket.handler.OnSessionConnected != nil {
			func() {
				defer OnError("")
				socket.handler.OnSessionConnected(session)
			}()
		}

		// async handle io
		go socket.handler.readIo(&socket.socket, session)
	}
}

func (socket *SocketAcceptor) Close() {

	socket.lock.Lock()
	defer socket.lock.Unlock()

	log.Printf("%s Close socket acceptor.", socket.addr)

	socket.closed = true

	if socket.handler.OnSessionClosed != nil {
		for _, session := range socket.sessions {
			session := session
			go func() {
				defer OnError("")
				socket.handler.OnSessionClosed(session)
			}()
		}
	}

	for i := 1; i <= 3; i++ {
		if err := socket.acceptor.Close(); err != nil {
			log.Printf("%s Close socket acceptor error. %s", socket.addr, err.Error())
			time.Sleep(1 * time.Second)
		} else {
			return
		}
	}
}

func (socket *SocketAcceptor) CloseSession(session *Session) {
	socket.removeSession(session.id)

	if socket.handler.OnSessionClosed != nil {
		go func() {
			defer OnError("")
			socket.handler.OnSessionClosed(session)
		}()
	}

	for i := 1; i <= 3; i++ {
		if err := session.connection.Close(); err != nil {
			if strings.Contains(err.Error(), "use of closed network") {
				return
			}
			log.Printf("[%d] Close socket connection %s error. %s", session.id, socket.addr, err.Error())
			time.Sleep(1 * time.Second)
		} else {
			return
		}
	}
}

func (socket *SocketAcceptor) addSession(session *Session) {
	socket.lock.Lock()
	defer socket.lock.Unlock()
	socket.sessions[session.id] = session
}

func (socket *SocketAcceptor) removeSession(sessionId int64) {
	socket.lock.Lock()
	defer socket.lock.Unlock()
	delete(socket.sessions, sessionId)
}

func (socket *SocketAcceptor) sendSessionId(connection net.Conn, sessionId int64) error {
	defer OnError("Send Session Id")
	time.Sleep(5 * time.Millisecond)
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, sessionId)
	if err != nil {
		log.Printf("Write session id error. %s", err.Error())
	}
	if _, ioErr := connection.Write(buf.Bytes()); ioErr != nil {
		return ioErr
	}
	return nil
}

func (socket *SocketAcceptor) SessionCount() int {
	return len(socket.sessions)
}

func OnError(msg string) {
	if r := recover(); r != nil {
		log.Printf("runtime error %s. %s\n%s", msg, r, string(debug.Stack()))
	}
}
