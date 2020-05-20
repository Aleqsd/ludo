package network

import "net"

type QueueEntry struct {
	QueueTime uint64
	DestAddr  *net.UDPAddr
	Msg       *NetplayMsg
}

func (q *QueueEntry) Init(time uint64, dst *net.UDPAddr, m *NetplayMsg) {
	q.QueueTime = time
	q.DestAddr = dst
	q.Msg = m
}
