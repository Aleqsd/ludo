package network

const (
	MAX_COMPRESSED_BITS = 4096
	MSG_MAX_PLAYERS     = 4
)

type MsgType int64

const (
	Invalid       MsgType = 0
	SyncRequest   MsgType = 1
	SyncReply     MsgType = 2
	Input         MsgType = 3
	QualityReport MsgType = 4
	QualityReply  MsgType = 5
	KeepAlive     MsgType = 6
	InputAck      MsgType = 7
)

type connectStatus struct {
	disconnected int64
	lastFrame    int64
}

type hdr struct {
	magic          int64
	sequenceNumber int64
	packetType     int64
}

type syncRequest struct {
	randomRequest   int64 /* please reply back with this random data */
	remoteMagic     int64
	remoteEndpoints int64
}
type syncReply struct {
	randomReply int64 /* OK, here's your random data back */
}

type qualityReport struct {
	frameAdvantage int64 /* what's the other guy's frame advantage? */
	ping           int64
}

type qualityReply struct {
	pong int64
}

type input struct {
	peerConnectStatus   connectStatus
	startFrame          int64
	disconnectRequested int64
	ackFrame            int64
	numBits             int64
	inputSize           int64                      // XXX: shouldn't be in every single packet!
	bits                [MAX_COMPRESSED_BITS]int64 /* must be last */
}

type inputAck struct {
	ackFrame int64
}
