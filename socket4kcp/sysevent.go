package socket4kcp

import (
	"reflect"

	"github.com/davyxu/cellnet"
	"github.com/davyxu/cellnet/proto/binary/coredef"
)

var (
	Meta_SessionConnected = cellnet.MessageMetaByName("coredef.SessionConnected")
	Meta_SessionAccepted  = cellnet.MessageMetaByName("coredef.SessionAccepted")
)

func systemEvent(ses cellnet.Session, e cellnet.EventType, hlist []cellnet.EventHandler) {

	ev := cellnet.NewSessionEvent(e, ses)

	var meta *cellnet.MessageMeta
	switch e {
	case cellnet.SessionEvent_Accepted:
		meta = Meta_SessionAccepted
	case cellnet.SessionEvent_Connected:
		meta = Meta_SessionConnected
	}

	ev.FromMeta(meta)

	cellnet.HandlerChainCall(hlist, ev)
}

func systemError(ses cellnet.Session, e cellnet.EventType, r cellnet.Result, hlist []cellnet.EventHandler) {

	ev := cellnet.NewSessionEvent(e, ses)

	// 直接放在这里, decoder里遇到系统事件不会进行decode操作
	switch e {
	case cellnet.SessionEvent_Closed:
		ev.Msg = &coredef.SessionClosed{Result: r}
	case cellnet.SessionEvent_AcceptFailed:
		ev.Msg = &coredef.SessionAcceptFailed{Result: r}
	case cellnet.SessionEvent_ConnectFailed:
		ev.Msg = &coredef.SessionConnectFailed{Result: r}
	default:
		panic("unknown system error")
	}

	ev.Type = e

	meta := cellnet.MessageMetaByType(reflect.TypeOf(ev.Msg))
	if meta != nil {
		ev.MsgID = meta.ID
	}

	cellnet.HandlerChainCall(hlist, ev)
}
