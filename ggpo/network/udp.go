package network

import (
	"github.com/libretro/ludo/ggpo/ggponet"
	"github.com/libretro/ludo/ggpo/lib"
)

type Udp struct {
	Callbacks ggponet.GGPOSessionCallbacks
	Poll      lib.Poll
	Port      uint16
}

func (u *Udp) Init(port uint16, poll lib.Poll, callbacks ggponet.GGPOSessionCallbacks) {
	u.Callbacks = callbacks
	u.Poll = poll
	//u.Poll.RegisterLoop(this)
	u.Port = port

	//Log("binding udp socket to port %d.\n", port);
}
