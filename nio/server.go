package nio

import (
	"bytes"
	"encoding/binary"
	"errors"
	"net"
	"ownx/log"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var logger = log.New()

// 客户端与服务端共用的结构
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

func CreateAcceptor(ip, port string, handler *Handler, codec Codec, ordered bool) (*SocketAcceptor, error) {
	if handler == nil {
		return nil, errors.New(ip + ":" + port + " Create socket acceptor failed. Handler is nil.")
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

func OnError(txt string) {
	if r := recover(); r != nil {
		logger.Error("Got a runtime error %s. %s\n%s", txt, r, string(debug.Stack()))
	}
}

func (socket *SocketAcceptor) Open() error {

	logger.Info("[server] Open socket %s acceptor and listening...", socket.addr)

	acceptor, err := net.Listen("tcp", socket.addr)
	if err != nil {
		logger.Error("[server] Open socket %s acceptor failed. %s", socket.addr, err.Error())
		return err
	}
	socket.acceptor = acceptor

	logger.Info("[server] Open socket %s acceptor success.", socket.addr)

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
			logger.Info("[server] Socket acceptor %s is closed. Normal finished.", socket.addr)
			return
		}
		connection, err := socket.acceptor.Accept()
		if err != nil {
			logger.Error("[server] [%s] Open socket connection failed. %s", socket.addr, err.Error())
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
			connection.Close()
			continue
		}

		logger.Info("[server] [%s] [%d] Accept connection from remote %s",
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

	logger.Info("[Close Event] [server] %s Close socket acceptor.", socket.addr)

	socket.closed = true

	// 异步回调
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
			logger.Error("[server] %s Close socket acceptor error. %s", socket.addr, err.Error())
			time.Sleep(1 * time.Second)
		} else {
			return
		}
	}
}

func (socket *SocketAcceptor) CloseSession(session *Session) {

	// 清除会话
	socket.removeSession(session.id)

	// 异步回调
	if socket.handler.OnSessionClosed != nil {
		go func() {
			defer OnError("")
			socket.handler.OnSessionClosed(session)
		}()
	}

	// 关闭socket
	for i := 1; i <= 3; i++ {
		if err := session.connection.Close(); err != nil {
			if strings.Contains(err.Error(), "use of closed network") {
				return
			}
			logger.Error("[client] [%d] Close socket connection %s error. %s", session.id, socket.addr, err.Error())
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
	binary.Write(buf, binary.BigEndian, sessionId)
	if _, ioErr := connection.Write(buf.Bytes()); ioErr != nil {
		return ioErr
	}
	return nil
}

func (socket *SocketAcceptor) SessionCount() int {
	return len(socket.sessions)
}
