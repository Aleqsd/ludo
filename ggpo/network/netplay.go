package network

import (
	"fmt"
	"net"

	"github.com/libretro/ludo/ggpo/ggponet"
	"github.com/libretro/ludo/ggpo/lib"
)

type Netplay struct {
	Callbacks     ggponet.GGPOSessionCallbacks
	Poll          lib.Poll
	Conn          net.Conn
	LocalPlayer   ggponet.GGPOPlayer
	HostingPlayer ggponet.GGPOPlayer
}

func (n *Netplay) Init(localPlayer ggponet.GGPOPlayer, hostingPlayer ggponet.GGPOPlayer /*, poll lib.Poll, callbacks ggponet.GGPOSessionCallbacks*/) {
	//n.Callbacks = callbacks
	//n.Poll = poll
	//n.Poll.RegisterLoop(n)
	n.LocalPlayer = localPlayer
	n.HostingPlayer = hostingPlayer

	//Log("binding udp socket to port %d.\n", port);
}

func (n *Netplay) SendInput(netoutput []byte) bool {
	if _, err := n.Conn.Write(netoutput[:]); err != nil {
		return false
	}
	return true
}

func (n *Netplay) ReadInput() {
	netinput := make([]byte, lib.GAMEINPUT_MAX_BYTES*lib.GAMEINPUT_MAX_PLAYERS)
	if _, err := n.Conn.Read(netinput[:]); err != nil {
		return
	}
	//TODO: Créer un channel pour stocker les inputs qui arrivent
	return
}

func (n *Netplay) HostConnection() bool {
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", n.LocalPlayer.IPAddress, int(n.LocalPlayer.Port)))
	if err != nil {
		return false
	}

	n.Conn, err = ln.Accept()
	if err != nil {
		return false
	}

	return true
}

func (n *Netplay) JoinConnection() bool {
	var err error
	n.Conn, err = net.Dial("tcp", fmt.Sprintf("%s:%d", n.LocalPlayer.IPAddress, int(n.LocalPlayer.Port)))
	if err != nil {
		return false
	}
	return true
}
