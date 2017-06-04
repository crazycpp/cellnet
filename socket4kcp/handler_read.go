package socket4kcp

import (
	"github.com/davyxu/cellnet"
	"time"
)

type ReadPacketHandler struct {
}

func (self *ReadPacketHandler) Call(ev *cellnet.SessionEvent) {

	switch ev.Type {
	case cellnet.SessionEvent_Recv:

		rawSes := ev.Ses.(*SocketSession)

		// 读超时
		read, _ := rawSes.FromPeer().SocketDeadline()

		if read != 0 {
			rawSes.stream.Raw().SetReadDeadline(time.Now().Add(read))
		}

		msgid, data, err := rawSes.stream.Read()

		if err != nil {

			recv, _ := rawSes.FromPeer().HandlerList()

			ev.SetResult(errToResult(err))

			systemError(ev.Ses, cellnet.SessionEvent_Closed, ev.Result(), recv)

		} else {

			ev.MsgID = msgid
			// 逻辑封包
			ev.Data = data
		}

	}

}

var defaultReadPacketHandler = new(ReadPacketHandler)

func StaticReadPacketHandler() cellnet.EventHandler {
	return defaultReadPacketHandler
}
