package nio

import (
	"bytes"
	"log"
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

const clientFlag string = "client"
const serverFlag string = "server"

func (handler *Handler) readIo(socket *socket, session *Session) {
	defer OnError("handler.readIo")

	flag := serverFlag
	if session.isClient {
		flag = clientFlag
	}
	conn := session.connection
	failTimes := 0
	lastMsg := &Msg{}
	buf := bytes.NewBuffer([]byte{})
	ioData := make([]byte, 1024*4)

	for {

		if socket.closed {
			log.Printf("%s [%s] [%d] Connection has closed. remote=%s. i will return.",
				socket.addr, flag, session.id, conn.RemoteAddr())
			return
		}

		rlen, err := conn.Read(ioData)

		if err != nil {
			// is remote conn closed?
			switch {
			case err.Error() == "EOF":
				log.Printf("%s [%s] [%d] Connection close. Cause of remote connection has closed. remote=%s",
					socket.addr, flag, session.id, conn.RemoteAddr())
				session.Close()
				return
			case strings.Contains(err.Error(), "closed by the remote host"):
				log.Printf("%s [%s] [%d] Connection close. Cause of remote connection has interrupted. remote=%s",
					socket.addr, flag, session.id, conn.RemoteAddr())
				session.Close()
				return
			case strings.Contains(err.Error(), "use of closed network"):
				log.Printf("%s [%s] [%d] Connection close. Cause of connection has closed. remote=%s",
					socket.addr, flag, session.id, conn.RemoteAddr())
				session.Close()
				return
			default:
			}

			failTimes++
			log.Printf("%s [%s] [%d] Read io error. times=%s. remote=%s. Cause of %s",
				socket.addr, flag, session.id, string(rune(failTimes)), conn.RemoteAddr(), err.Error())

			if socket.handler.OnException != nil {
				go func() {
					defer OnError("")
					socket.handler.OnException(session)
				}()
			}

			if failTimes >= 3 {
				log.Printf("%s [%s] [%d] Connection close. remote=%s. Error finished.",
					socket.addr, flag, session.id, conn.RemoteAddr())
				session.Close()
				return
			}

			time.Sleep(500 * time.Millisecond)
			continue

		} else {
			failTimes = 0
		}

		if rlen > 0 {
			buf.Write(ioData[0:rlen])

			for {

				if ok := socket.handler.codec.Decode(lastMsg, buf); ok {

					// copy msg
					cpMsg := *lastMsg

					if socket.handler.OnMessageReceived != nil {
						if socket.ordered {
							socket.handler.OnMessageReceived(session, &cpMsg)
						} else {
							go func(cpMsg *Msg) {
								defer OnError("")
								socket.handler.OnMessageReceived(session, cpMsg)
							}(&cpMsg)
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
