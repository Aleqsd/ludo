package network

import (
	"encoding/binary"
	"fmt"
	"net"

	"github.com/libretro/ludo/ggpo/ggponet"
	"github.com/libretro/ludo/ggpo/lib"
)

type Event struct {
	Input     lib.GameInput
	PlayerNum int64
}

type Netplay struct {
	Callbacks     ggponet.GGPOSessionCallbacks
	Poll          lib.Poll
	Conn          *net.UDPConn
	LocalAddr     *net.UDPAddr
	RemoteAddr    *net.UDPAddr
	Queue         int64
	IsHosting     bool
}

func (n *Netplay) Init(remotePlayer ggponet.GGPOPlayer, queue int64 /*, poll lib.Poll, callbacks ggponet.GGPOSessionCallbacks*/) {
	//n.Callbacks = callbacks
	//n.Poll = poll
	//n.Poll.RegisterLoop(n)
	n.LocalAddr, _ = net.ResolveUDPAddr("udp4", "127.0.0.1:8089")
	n.RemoteAddr, _ = net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", remotePlayer.IPAddress, int(remotePlayer.Port)))
	n.Queue = queue

	//Log("binding udp socket to port %d.\n", port);
}

func (n *Netplay) Write(netoutput []byte) {
	var err error
	if n.IsHosting {
		_, err = n.Conn.WriteToUDP(netoutput, n.RemoteAddr)
	} else {
		_, err = n.Conn.Write(netoutput)
	}
	if err != nil {
		fmt.Println(err)
		return
	}
}

func (n *Netplay) Read() {	
	for {
		netinput := make([]byte, lib.GAMEINPUT_MAX_BYTES*lib.GAMEINPUT_MAX_PLAYERS)
		n, _, err := n.Conn.ReadFromUDP(netinput)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Printf(string(netinput[0:n]))
		//TODO: Créer un channel pour stocker les inputs qui arrivent
	}
}

func (n *Netplay) SendInput(input lib.GameInput) {
	inputByte := n.InputToByte(input)
	n.Write(inputByte)
}

func (n *Netplay) ReceiveInput() Event {
	//TODO: get channel value
	//Convert it to event via ByteToEvent() function
	//Return the event
	return Event{}
}

func (n *Netplay) InputToByte(input lib.GameInput) []byte {
	frameByte := make([]byte, 8)
	binary.LittleEndian.PutUint64(frameByte, uint64(input.Frame))
	inputByte := make([]byte, len(input.Bits)+len(frameByte))
	count := 0
	for i := 0; i < len(frameByte); i++ {
		inputByte[count] = frameByte[i]
		count++
	}
	for i := 0; i < len(input.Bits); i++ {
		inputByte[count] = input.Bits[i]
		count++
	}
	return inputByte
}

func (n *Netplay) ByteToInput(inputByte []byte) lib.GameInput {
	input := lib.GameInput{}
	count := 0
	frameByte := make([]byte, 8)
	for i := 0; i < len(frameByte); i++ {
		frameByte[i] = inputByte[count]
		count++
	}
	bits := make([]byte, len(inputByte)-len(frameByte))
	for i := 0; i < len(bits); i++ {
		bits[i] = inputByte[count]
		count++
	}

	frame := int64(binary.LittleEndian.Uint64(frameByte))
	input.Frame = frame
	input.Bits = bits

	return input
}

func (n *Netplay) HostConnection() {
	n.IsHosting = true
	var err error
	n.Conn, err = net.ListenUDP("udp4", n.LocalAddr)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer n.Conn.Close()
	go n.Read()
}

func (n *Netplay) JoinConnection() {
	n.IsHosting = false
	var err error
	n.Conn, err = net.DialUDP("udp4", n.LocalAddr, n.RemoteAddr)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer n.Conn.Close()
	go n.Read()
}

func (n *Netplay) Disconnect() ggponet.GGPOErrorCode {
	n.Conn.Close()
	if n.Conn == nil {
		return ggponet.GGPO_OK
	}
	return ggponet.GGPO_ERRORCODE_PLAYER_DISCONNECTED
}
