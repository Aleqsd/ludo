package lib

import (
	"math"
	"time"
)

const (
	MAX_POLLABLE_HANDLES = 64
)

type Poll struct {
	StartTime   int64
	HandleCount int64
	//Handles [MAX_POLLABLE_HANDLES]HANDLE
	HandleSinks [MAX_POLLABLE_HANDLES]PollSinkCb
	
	MsgSinks [16]PollSinkCb
	LoopSinks [16]PollSinkCb
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
	//PollSinkCb() : sink(NULL), cookie(NULL) { }
	//PollSinkCb(IPollSink *s, void *c) : sink(s), cookie(c) { }
}

type PollPeriodicSinkCb : PollSinkCb {
	Interval int64
	LastFired int64
	//PollPeriodicSinkCb() : PollSinkCb(NULL, NULL), interval(0), last_fired(0) { }
	//PollPeriodicSinkCb(IPollSink *s, void *c, int i) : PollSinkCb(s, c), interval(i), last_fired(0) { }
}

func (p *Poll) Init() {
	p.HandleCount = 0
	p.StartTime = 0

	//_handles[_handle_count++] = CreateEvent(NULL, true, false, NULL);

}

func (p *Poll) RegisterHandle() {
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

	// res = WaitForMultipleObjects(_handle_count, _handles, false, timeout);
	// if (res >= WAIT_OBJECT_0 && res < WAIT_OBJECT_0 + _handle_count) {
	//    i = res - WAIT_OBJECT_0;
	//    finished = !_handle_sinks[i].sink->OnHandlePoll(_handle_sinks[i].cookie) || finished;
	// }

	for i := 0; i < len(MsgSinks) ; i++ {
		PollSinkCb &cb = MsgSinks[i]
		finished = !cb.Sink.OnMsgPoll(cb.Cookie) || finished
	}

	for i := 0; i < len(PeriodicSinks) ; ++ {
		PollPeriodicSinkCb &cb = PeriodicSinks[i]
		if cb.Interval + cb.LastFired <= elapsed {
			cb.LastFired = (elapsed/cb.Interval) * cb.Interval
			finished = !cb.Sink.OnPeriodicPoll(cb.Cookie, cb.LastFired) || finished
		}
		
	}

	for i := 0; i < len(LoopSinks) ; ++ {
		PollSinkCb &cb = LoopSinks[i]
		finished = !cb.Sink.OnLoopPoll(cb.Cookie) || finished
	}

	return finished
}

func (p *Poll) ComputeWaitTime(elapsed int64) int64 {
	var waitTime int64 = math.MaxInt64
	count := len(PeriodicSinks)

	if count > 0 {
		for i := 0; i < count ; i++ {
			PollPeriodicSinkCb &cb = PeriodicSinks[i]
			timeout := (cb.Interval + cb.LastFired) - elapsed
			if waitTime == MaxInt64 || timeout < waitTime {
				waitTime = MAX(timeout, 0)
			}
		}
	}
	return waitTime
}
