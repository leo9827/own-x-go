package nio

import (
	"bytes"
	"strings"
	"time"
)

type Handler struct {
	codec              Codec
	OnSessionConnected func(session *Session)
	OnSessionClosed    func(session *Session)
	OnMessageReceived  func(session *Session, message *Msg)
	OnException        func(session *Session)
}

func (handler *Handler) readIo(socket *socket, session *Session) {

	defer OnError("handler.readIo")

	flag := "server"
	if session.isClient {
		flag = "client"
	}

	conn := session.connection

	failTimes := 0

	lastMsg := &Msg{}

	buf := bytes.NewBuffer([]byte{})

	// 每次读入的新数据
	ioData := make([]byte, 1024*4)
	for {

		if socket.closed {
			logger.Error("%s [%s] [%d] Connection has closed. remote=%s. i will return.", socket.addr, flag, session.id, conn.RemoteAddr())
			return
		}

		rlen, err := conn.Read(ioData)

		if err != nil {

			// remote conn is closed?
			if err.Error() == "EOF" {
				logger.Info("%s [%s] [%d] Connection close. Cause of remote connection has closed. remote=%s", socket.addr, flag, session.id, conn.RemoteAddr())
				session.Close()
				return
			}
			if strings.Contains(err.Error(), "closed by the remote host") {
				logger.Info("%s [%s] [%d] Connection close. Cause of remote connection has interrupted. remote=%s", socket.addr, flag, session.id, conn.RemoteAddr())
				session.Close()
				return
			}
			if strings.Contains(err.Error(), "use of closed network") {
				logger.Info("%s [%s] [%d] Connection close. Cause of connection has closed. remote=%s", socket.addr, flag, session.id, conn.RemoteAddr())
				session.Close()
				return
			}

			// 错误后重试3次，3次后还错误则关闭连接
			failTimes++
			logger.Error("%s [%s] [%d] Read io error. times=%s. remote=%s. Cause of %s", socket.addr, flag, session.id, string(rune(failTimes)), conn.RemoteAddr(), err.Error())
			if socket.handler.OnException != nil {
				go func() {
					defer OnError("")
					socket.handler.OnException(session)
				}()
			}
			if failTimes >= 3 {
				logger.Error("%s [%s] [%d] Connection close. remote=%s. Error finished.", socket.addr, flag, session.id, conn.RemoteAddr())
				session.Close()
				return
			}
			time.Sleep(200 * time.Millisecond)
			continue
		} else {
			failTimes = 0
		}

		if rlen > 0 {

			buf.Write(ioData[0:rlen])

			for {

				if ok := socket.handler.codec.Decode(lastMsg, buf); ok {

					// copy msg
					copyMsg := *lastMsg

					if socket.handler.OnMessageReceived != nil {
						if socket.ordered {
							socket.handler.OnMessageReceived(session, &copyMsg)
						} else {
							go func(cmsg *Msg) {
								defer OnError("")
								socket.handler.OnMessageReceived(session, cmsg)
							}(&copyMsg)
						}
					}

					// re init
					lastMsg = &Msg{}

				} else {
					break
				}
			}
		}
	}
}
