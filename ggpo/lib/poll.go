package lib

import (
	"math"

	"github.com/libretro/ludo/ggpo/platform"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/windows"
)

const (
	MAX_POLLABLE_HANDLES = 64
)

type Poll struct {
	StartTime   uint64
	HandleCount int64
	Handles     []windows.Handle
	HandleSinks [MAX_POLLABLE_HANDLES]PollSinkCb

	MsgSinks      StaticBuffer
	LoopSinks     StaticBuffer
	PeriodicSinks StaticBuffer
}

type IPollSink interface {
	OnHandlePoll(cookie []byte) bool
	OnMsgPoll(cookie []byte) bool
	OnPeriodicPoll(cookie []byte, lastFired int64) bool
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
	var err error
	p.HandleCount = 0
	p.StartTime = 0

	p.MsgSinks.Init(16)
	p.LoopSinks.Init(16)
	p.PeriodicSinks.Init(16)
	p.Handles = make([]windows.Handle, MAX_POLLABLE_HANDLES)
	p.Handles[p.HandleCount], err = windows.CreateEvent(nil, 1, 0, nil)
	p.HandleCount++
	if err != nil {
		logrus.Panic("Assert error on CreateEvent")
	}

}

func (p *Poll) RegisterHandle(sink *IPollSink, h windows.Handle, cookie []byte) {
	if p.HandleCount >= MAX_POLLABLE_HANDLES-1 {
		logrus.Panic("Assert error on HandleCount too high")
	}

	p.Handles[p.HandleCount] = h
	var pollSink PollSinkCb
	pollSink.Init(sink, cookie)
	p.HandleSinks[p.HandleCount] = pollSink
	p.HandleCount++
}

func (p *Poll) RegisterMsgLoop(sink *IPollSink, cookie []byte) {
	var pollSink PollSinkCb
	pollSink.Init(sink, cookie)
	var u U = &pollSink
	p.MsgSinks.PushBack(&u)
}

func (p *Poll) RegisterLoop(sink *IPollSink, cookie []byte) {
	var pollSink PollSinkCb
	pollSink.Init(sink, cookie)
	var u U = &pollSink
	p.MsgSinks.PushBack(&u)
}

func (p *Poll) RegisterPeriodic(sink *IPollSink, interval int64, cookie []byte) {
	var pollPeriodicSink PollPeriodicSinkCb
	pollPeriodicSink.Init(sink, cookie, interval)
	var u U = &pollPeriodicSink
	p.MsgSinks.PushBack(&u)
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

	res, err := windows.WaitForMultipleObjects(p.Handles, false, uint32(timeout))
	if err != nil {
		logrus.Panic("Assert error on WaitForMultipleObjects")
	}
	if res >= windows.WAIT_OBJECT_0 && int64(res) < int64(windows.WAIT_OBJECT_0)+p.HandleCount {
		j := res - windows.WAIT_OBJECT_0
		var s IPollSink = *p.HandleSinks[j].Sink
		finished = !s.OnHandlePoll(p.HandleSinks[j].Cookie) || finished
	}

	var i int64
	for i = 0; i < p.MsgSinks.Size; i++ {
		var cb PollSinkCb = p.MsgSinks.Get(i).(PollSinkCb)
		var s IPollSink = *cb.Sink
		finished = !s.OnMsgPoll(cb.Cookie) || finished
	}

	for i = 0; i < p.PeriodicSinks.Size; i++ {
		var cb PollPeriodicSinkCb
		cb.PollSinkCbVal = p.PeriodicSinks.Get(i).(PollSinkCb)
		if cb.Interval+cb.LastFired <= int64(elapsed) {
			cb.LastFired = (int64(elapsed) / cb.Interval) * cb.Interval
			var s IPollSink = *cb.PollSinkCbVal.Sink
			finished = !s.OnPeriodicPoll(cb.PollSinkCbVal.Cookie, cb.LastFired) || finished
		}

	}

	for i = 0; i < p.LoopSinks.Size; i++ {
		var cb PollSinkCb = p.LoopSinks.Get(i).(PollSinkCb)
		var s IPollSink = *cb.Sink
		finished = !s.OnLoopPoll(cb.Cookie) || finished
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
