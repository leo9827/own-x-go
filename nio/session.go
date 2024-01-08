package nio

import (
	"errors"
	"net"
)

type Session struct {
	id         int64
	isClient   bool
	attribute  map[string]interface{} // local attributes
	codec      Codec
	acceptor   *SocketAcceptor
	connector  *SocketConnector
	connection net.Conn
}

func (session *Session) GetSessionId() int64 {
	return session.id
}

func (session *Session) SetAttribute(k string, v interface{}) {
	if session.attribute == nil {
		session.attribute = make(map[string]interface{})
	}
	session.attribute[k] = v
}

func (session *Session) GetAttribute(k string) interface{} {
	if session.attribute == nil {
		return nil
	}
	return session.attribute[k]
}

func (session *Session) WriteMsg(msg *Msg) error {
	if session.connection == nil {
		return errors.New("not connected")
	}
	if _, err := session.connection.Write(session.codec.Encode(msg).Bytes()); err != nil {
		return err
	}
	return nil
}

func (session *Session) Write(flag uint8, data interface{}) error {
	if session.connection == nil {
		return errors.New("not connected")
	}
	d := session.codec.Encode(CreateMsg(flag, data)).Bytes()
	if _, err := session.connection.Write(d); err != nil {
		return err
	}
	return nil
}

func (session *Session) GetLocalAddr() net.Addr {
	if session.connection != nil {
		return session.connection.LocalAddr()
	}
	return nil
}

func (session *Session) GetRemoteAddr() net.Addr {
	if session.connection != nil {
		return session.connection.RemoteAddr()
	}
	return nil
}

func (session *Session) Close() {

	if session.isClient {
		session.connector.Close()
	} else {
		session.acceptor.CloseSession(session)
	}
}
