package socket4kcp

import (
	"sync"

	"github.com/davyxu/cellnet"
	"time"
)

type SocketSession struct {
	OnClose func() // 关闭函数回调

	id int64

	p cellnet.Peer

	endSync sync.WaitGroup

	needNotifyWrite bool // 是否需要通知写线程关闭

	// handler相关上下文
	stream cellnet.PacketStream

	sendList *eventList
}

func (self *SocketSession) ID() int64 {
	return self.id
}

func (self *SocketSession) FromPeer() cellnet.Peer {
	return self.p
}

func (self *SocketSession) Close() {
	self.sendList.Add(nil)
}

func (self *SocketSession) Send(data interface{}) {

	ev := cellnet.NewSessionEvent(cellnet.SessionEvent_Send, self)
	ev.Msg = data

	self.RawSend(ev.SendHandler, ev)

}

func (self *SocketSession) RawSend(sendHandler []cellnet.EventHandler, ev *cellnet.SessionEvent) {

	if sendHandler == nil {
		_, sendHandler = self.p.HandlerList()
	}

	ev.Ses = self

	cellnet.HandlerChainCall(sendHandler, ev)
}

func (self *SocketSession) Post(data interface{}) {

	ev := cellnet.NewSessionEvent(cellnet.SessionEvent_Post, self)

	ev.Msg = data

	MsgLog(ev)

	self.p.Call(ev)
}

func (self *SocketSession) RawPost(recvHandler []cellnet.EventHandler, ev *cellnet.SessionEvent) {
	if recvHandler == nil {
		recvHandler, _ = self.p.HandlerList()
	}

	ev.Ses = self

	cellnet.HandlerChainCall(recvHandler, ev)
}

// 发送线程
func (self *SocketSession) sendThread() {

	var writeList []*cellnet.SessionEvent

	for {

		willExit := false
		writeList = writeList[0:0]

		// 复制出队列
		packetList := self.sendList.BeginPick()

		for _, ev := range packetList {

			if ev == nil {
				willExit = true
				break
			} else {
				writeList = append(writeList, ev)
			}
		}

		self.sendList.EndPick()

		// 写超时
		_, write := self.FromPeer().SocketDeadline()

		if write != 0 {
			self.stream.Raw().SetWriteDeadline(time.Now().Add(write))
		}

		// 写队列
		for _, ev := range writeList {

			if err := self.stream.Write(ev.MsgID, ev.Data); err != nil {
				willExit = true
				break
			}

		}

		if err := self.stream.Flush(); err != nil {
			willExit = true
		}

		if willExit {
			goto exitsendloop
		}
	}

exitsendloop:

	// 不需要读线程再次通知写线程
	self.needNotifyWrite = false

	// 关闭socket,触发读错误, 结束读循环
	self.stream.Close()

	// 通知发送线程ok
	self.endSync.Done()
}

func (self *SocketSession) recvThread() {

	recv, _ := self.p.HandlerList()

	for {

		ev := cellnet.NewSessionEvent(cellnet.SessionEvent_Recv, self)

		cellnet.HandlerChainCall(recv, ev)

		if ev.Result() != cellnet.Result_OK {
			break
		}

	}

	if self.needNotifyWrite {
		self.Close()
	}

	// 通知接收线程ok
	self.endSync.Done()
}

func (self *SocketSession) run() {
	// 布置接收和发送2个任务
	// bug fix感谢viwii提供的线索
	self.endSync.Add(2)

	go func() {

		// 等待2个任务结束
		self.endSync.Wait()

		// 在这里断开session与逻辑的所有关系
		if self.OnClose != nil {
			self.OnClose()
		}
	}()

	// 接收线程
	go self.recvThread()

	// 发送线程
	go self.sendThread()
}

func newSession(composer cellnet.PacketStream, p cellnet.Peer) *SocketSession {

	self := &SocketSession{
		stream:          composer,
		p:               p,
		needNotifyWrite: true,
		sendList:        NewPacketList(),
	}

	// 使用peer的统一设置
	if s, ok := self.stream.(*TLVStream); ok {
		s.SetMaxPacketSize(p.MaxPacketSize())
	}

	return self
}
