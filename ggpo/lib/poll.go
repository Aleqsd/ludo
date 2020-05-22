package lib

type Poll struct {
	LoopSinks StaticBuffer
}

type IPollSink interface {
	OnLoopPoll(cookie []byte) bool
}

type PollSinkCb struct {
	Sink   *IPollSink
	Cookie []byte
}

func (p *PollSinkCb) Init(s *IPollSink, c []byte) {
	p.Sink = s
	p.Cookie = c
}

func (p *Poll) Init() {
	p.LoopSinks.Init(16)
}

func (p *Poll) RegisterLoop(sink *IPollSink, cookie []byte) {
	var pollSink PollSinkCb
	pollSink.Init(sink, cookie)
	var u U = &pollSink
	p.LoopSinks.PushBack(&u)
}

func (p *Poll) Pump() bool {
	finished := false
	var i int64
	for i = 0; i < p.LoopSinks.Size; i++ {
		var cb PollSinkCb = p.LoopSinks.Get(i).(PollSinkCb)
		var s IPollSink = *cb.Sink
		finished = !s.OnLoopPoll(cb.Cookie) || finished
	}

	return finished
}
