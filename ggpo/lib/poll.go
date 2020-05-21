package lib

import (
	"math"

	"github.com/libretro/ludo/ggpo/platform"
	"github.com/sirupsen/logrus"
)

type HANDLE int

const (
	MAX_POLLABLE_HANDLES = 64
)

type Poll struct {
	StartTime   uint64
	HandleCount int64
	Handles     [MAX_POLLABLE_HANDLES]HANDLE
	HandleSinks [MAX_POLLABLE_HANDLES]PollSinkCb

	MsgSinks      StaticBuffer
	LoopSinks     StaticBuffer
	PeriodicSinks StaticBuffer
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

	p.MsgSinks.Init(16)
	p.LoopSinks.Init(16)
	p.PeriodicSinks.Init(16)
	//p.Handles[p.HandleCount++] = CreateEvent(NULL, true, false, NULL)

}

func (p *Poll) RegisterHandle(sink *IPollSink, h HANDLE, cookie []byte) {

	if p.HandleCount >= MAX_POLLABLE_HANDLES-1 {
		logrus.Panic("Assert error on HandleCount too high")
	}

	p.Handles[p.HandleCount] = h
	var pollSink PollSinkCb
	pollSink.Init(sink, cookie)
	p.HandleSinks[p.HandleCount] = pollSink
	p.HandleCount++
}

//TODO: Here, cf commented lines those functions
func (p *Poll) RegisterMsgLoop(sink *IPollSink, cookie []byte) {
	var pollSink PollSinkCb
	pollSink.Init(sink, cookie)
	//p.MsgSinks.PushBack(pollSink)
}

func (p *Poll) RegisterLoop(sink *IPollSink, cookie []byte) {
	var pollSink PollSinkCb
	pollSink.Init(sink, cookie)
	//p.MsgSinks.PushBack(pollSink)
}

func (p *Poll) RegisterPeriodic(sink *IPollSink, interval int64, cookie []byte) {
	var pollPeriodicSink PollPeriodicSinkCb
	pollPeriodicSink.Init(sink, cookie, interval)
	//p.MsgSinks.PushBack(pollPeriodicSink)
}

func (p *Poll) Run() {
	for p.Pump(100) {
		continue
	}
}

func (p *Poll) Pump(timeout int64) bool {
	finished := false

	if p.StartTime == 0 {
		p.StartTime = platform.GetCurrentTimeMS()
	}

	elapsed := platform.GetCurrentTimeMS() - p.StartTime
	maxWait := p.ComputeWaitTime(elapsed)
	if maxWait != math.MaxInt64 {
		timeout = MIN(int64(timeout), int64(maxWait))
	}

	//TODO: WaitForMultipleObjects ?
	//var res int64
	// res = WaitForMultipleObjects(_handle_count, _handles, false, timeout);
	// if (res >= WAIT_OBJECT_0 && res < WAIT_OBJECT_0 + _handle_count) {
	//    i = res - WAIT_OBJECT_0;
	//    finished = !_handle_sinks[i].sink->OnHandlePoll(_handle_sinks[i].cookie) || finished;
	// }

	//TODO: Here, cf commented lines here
	var i int64
	for i = 0; i < p.MsgSinks.Size; i++ {
		//var cb PollSinkCb = p.MsgSinks.Get(i).(PollSinkCb)
		//finished = !cb.Sink.OnMsgPoll(cb.Cookie) || finished
	}

	for i = 0; i < p.PeriodicSinks.Size; i++ {
		var cb PollPeriodicSinkCb
		cb.PollSinkCbVal = p.PeriodicSinks.Get(i).(PollSinkCb)
		if cb.Interval+cb.LastFired <= int64(elapsed) {
			//cb.LastFired = (elapsed / cb.Interval) * cb.Interval
			//finished = !cb.Sink.OnPeriodicPoll(cb.PollSinkCbVal.Cookie, cb.LastFired) || finished
		}

	}

	for i = 0; i < p.LoopSinks.Size; i++ {
		//var cb PollSinkCb = p.LoopSinks.Get(i).(PollSinkCb)
		//finished = !cb.Sink.OnLoopPoll(cb.Cookie) || finished
	}

	return finished
}

func (p *Poll) ComputeWaitTime(elapsed uint64) uint64 {
	var waitTime uint64 = math.MaxUint64
	count := p.PeriodicSinks.Size

	var i int64
	if count > 0 {
		for i = 0; i < count; i++ {
			var cb PollPeriodicSinkCb
			cb.PollSinkCbVal = p.PeriodicSinks.Get(i).(PollSinkCb)
			timeout := uint64(cb.Interval+cb.LastFired) - elapsed
			if waitTime == math.MaxInt64 || timeout < waitTime {
				waitTime = uint64(MAX(int64(timeout), 0))
			}
		}
	}
	return waitTime
}
