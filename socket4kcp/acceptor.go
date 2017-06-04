package socket4kcp

import (
	"net"

	"github.com/davyxu/cellnet"
	"github.com/xtaci/kcp-go"
)

type socketAcceptor struct {
	*peerBase
	*sessionMgr

	listener net.Listener

	running bool
}

func (self *socketAcceptor) Start(address string) cellnet.Peer {

	self.address = address

	//ln, err := net.Listen("tcp", address)
	ln, err := kcp.Listen(address)

	self.listener = ln

	if err != nil {

		log.Errorf("#listen failed(%s) %v", self.nameOrAddress(), err.Error())
		return self
	}

	self.running = true

	log.Infof("#listen(%s) %s", self.name, self.address)

	// 接受线程
	go func() {
		for self.running {
			conn, err := ln.Accept()

			if err != nil {
				log.Errorf("#accept failed(%s) %v", self.nameOrAddress(), err.Error())

				systemError(nil, cellnet.SessionEvent_AcceptFailed, errToResult(err), self.safeRecvHandler())

				break
			}

			// 处理连接进入独立线程, 防止accept无法响应
			go func() {

				ses := newSession(self.genPacketStream(conn), self)

				// 添加到管理器
				self.sessionMgr.Add(ses)

				// 断开后从管理器移除
				ses.OnClose = func() {
					self.sessionMgr.Remove(ses)
				}

				ses.run()

				// 通知逻辑
				systemEvent(ses, cellnet.SessionEvent_Accepted, self.safeRecvHandler())
			}()

		}

	}()

	return self
}

func (self *socketAcceptor) Stop() {

	if !self.running {
		return
	}

	self.running = false

	self.listener.Close()
}

func NewAcceptor(evq cellnet.EventQueue) cellnet.Peer {

	self := &socketAcceptor{
		sessionMgr: newSessionManager(),
		peerBase:   newPeerBase(evq),
	}

	return self
}
