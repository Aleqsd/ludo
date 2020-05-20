package lib

import (
	"math"
	"time"
)

type HANDLE int

const (
	MAX_POLLABLE_HANDLES = 64
)

type Poll struct {
	StartTime   int64
	HandleCount int64
	Handles     [MAX_POLLABLE_HANDLES]HANDLE
	HandleSinks [MAX_POLLABLE_HANDLES]PollSinkCb

	MsgSinks      [16]PollSinkCb
	LoopSinks     [16]PollSinkCb
	PeriodicSinks [16]PollSinkCb
}

type IPollSink interface {
	OnHandlePoll(cookie []byte)
	OnMsgPoll(cookie []byte)
	OnPeriodicPoll(cookie []byte, lastFired int64)
	OnLoopPoll(cookie []byte)
}

type PollSinkCb struct {
	Sink   *IPollSink
	Cookie []byte
}

func (p *PollSinkCb) Init(s *IPollSink, c []byte) {
	p.Sink = s
	p.Cookie = c
}

type PollPeriodicSinkCb struct {
	PollSinkCbVal PollSinkCb
	Interval      int64
	LastFired     int64
}

func (p *PollPeriodicSinkCb) Init(s *IPollSink, c []byte, i int64) {
	p.PollSinkCbVal.Init(s, c)
	p.Interval = i
	p.LastFired = 0
}

func (p *Poll) Init() {
	p.HandleCount = 0
	p.StartTime = 0

	//p.Handles[p.HandleCount++] = CreateEvent(NULL, true, false, NULL)

}

func (p *Poll) RegisterHandle(s *IPollSink, h HANDLE, cookie []byte) {
	p.Handles[p.HandleCount] = h
}

func (p *Poll) RegisterMsgLoop() {
}

func (p *Poll) RegisterLoop() {
}

func (p *Poll) RegisterPeriodic() {
}

func (p *Poll) Run() {
}

func (p *Poll) Pump(timeout int64) bool {
	//var res int64
	finished := false

	if p.StartTime == 0 {
		p.StartTime = time.Now().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond))
	}

	elapsed := time.Now().UnixNano()/(int64(time.Millisecond)/int64(time.Nanosecond)) - p.StartTime
	maxWait := p.ComputeWaitTime(elapsed)
	if maxWait != math.MaxInt64 {
		timeout = MIN(timeout, maxWait)
	}

	//TODO WaitForMultipleObjects ?
	// res = WaitForMultipleObjects(_handle_count, _handles, false, timeout);
	// if (res >= WAIT_OBJECT_0 && res < WAIT_OBJECT_0 + _handle_count) {
	//    i = res - WAIT_OBJECT_0;
	//    finished = !_handle_sinks[i].sink->OnHandlePoll(_handle_sinks[i].cookie) || finished;
	// }

	for i := 0; i < len(p.MsgSinks); i++ {
		var cb PollSinkCb = p.MsgSinks[i]
		finished = !cb.Sink.OnMsgPoll(cb.Cookie) || finished
	}

	for i := 0; i < len(p.PeriodicSinks); i++ {
		var cb PollPeriodicSinkCb
		cb.PollSinkCbVal = p.PeriodicSinks[i]
		if cb.Interval+cb.LastFired <= elapsed {
			cb.LastFired = (elapsed / cb.Interval) * cb.Interval
			finished = !cb.Sink.OnPeriodicPoll(cb.PollSinkCbVal.Cookie, cb.LastFired) || finished
		}

	}

	for i := 0; i < len(p.LoopSinks); i++ {
		var cb PollSinkCb = p.LoopSinks[i]
		finished = !cb.Sink.OnLoopPoll(cb.Cookie) || finished
	}

	return finished
}

func (p *Poll) ComputeWaitTime(elapsed int64) int64 {
	var waitTime int64 = math.MaxInt64
	count := len(p.PeriodicSinks)

	if count > 0 {
		for i := 0; i < count; i++ {
			var cb PollPeriodicSinkCb
			cb.PollSinkCbVal = p.PeriodicSinks[i]
			timeout := (cb.Interval + cb.LastFired) - elapsed
			if waitTime == math.MaxInt64 || timeout < waitTime {
				waitTime = MAX(timeout, 0)
			}
		}
	}
	return waitTime
}
