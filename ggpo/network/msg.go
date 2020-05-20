package network

import (
	"unsafe"

	"github.com/libretro/ludo/ggpo/ggponet"
)

const (
	MAX_COMPRESSED_BITS = 4096
	MSG_MAX_PLAYERS     = 4
)

type MsgType int64

const (
	Invalid MsgType = iota
	SyncRequest
	SyncReply
	Input
	QualityReport
	QualityReply
	KeepAlive
	InputAck
)

type hdr struct {
	Magic          int64
	SequenceNumber int64
	Type           MsgType
}

type syncRequest struct {
	RandomRequest   int64 /* please reply back with this random data */
	RemoteMagic     int64
	RemoteEndpoints int64
}
type syncReply struct {
	RandomReply int64 /* OK, here's your random data back */
}

type qualityReport struct {
	FrameAdvantage int64 /* what's the other guy's frame advantage? */
	Ping           int64
}

type qualityReply struct {
	Pong int64
}

type input struct {
	PeerConnectStatus   []ggponet.ConnectStatus
	StartFrame          int64
	DisconnectRequested bool
	AckFrame            int64
	NumBits             int64
	InputSize           int64  // XXX: shouldn't be in every single packet!
	Bits                []byte /* must be last */
}

type inputAck struct {
	AckFrame int64
}

type NetplayMsg struct {
	ConnectStatus ggponet.ConnectStatus
	Hdr           hdr
	SyncRequest   syncRequest
	SyncReply     syncReply
	QualityReport qualityReport
	QualityReply  qualityReply
	Input         input
	InputAck      inputAck
}

func (n *NetplayMsg) Init(t MsgType) {
	n.Hdr.Type = t
}

func (n *NetplayMsg) PacketSize() int64 {
	return int64(unsafe.Sizeof(n.Hdr)) + n.PayloadSize()
}

func (n *NetplayMsg) PayloadSize() int64 {
	var size int64

	switch n.Hdr.Type {
	case SyncRequest:
		return int64(unsafe.Sizeof(n.SyncRequest))
	case SyncReply:
		return int64(unsafe.Sizeof(n.SyncReply))
	case QualityReport:
		return int64(unsafe.Sizeof(n.QualityReport))
	case QualityReply:
		return int64(unsafe.Sizeof(n.QualityReply))
	case InputAck:
		return int64(unsafe.Sizeof(n.InputAck))
	case KeepAlive:
		return 0
	case Input:
		size = int64(unsafe.Sizeof(n.Input))
		size += (n.Input.NumBits + 7) / 8
		return size
	}
	return 0
}
