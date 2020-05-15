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

func (n *Netplay) Write(netoutput []byte) bool {
	if _, err := n.Conn.Write(netoutput[:]); err != nil {
		return false
	}
	return true
}

func (n *Netplay) Read() {
	for {
		netinput := make([]byte, lib.GAMEINPUT_MAX_BYTES*lib.GAMEINPUT_MAX_PLAYERS)
		if _, err := n.Conn.Read(netinput[:]); err != nil {
			return
		}
		//TODO: Créer un channel pour stocker les inputs qui arrivent
	}
}

func (n *Netplay) SendInput(input lib.GameInput) bool {
	inputByte := n.InputToByte(input)
	return n.Write(inputByte)
}

func (n *Netplay) ReceiveInput() Event {
	//TODO: get channel value
	//Convert it to event via ByteToEvent() function
	//Return the event
	return Event{}
}

func (n *Netplay) InputToByte(input lib.GameInput) []byte {
	playerNumByte := make([]byte, 8)
	binary.LittleEndian.PutUint64(playerNumByte, uint64(n.LocalPlayer.PlayerNum))
	frameByte := make([]byte, 8)
	binary.LittleEndian.PutUint64(frameByte, uint64(input.Frame))
	inputByte := make([]byte, len(input.Bits)+len(playerNumByte)+len(frameByte))
	count := 0
	for i := 0; i < len(playerNumByte); i++ {
		inputByte[count] = playerNumByte[i]
		count++
	}
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

func (n *Netplay) ByteToEvent(inputByte []byte) Event {
	input := lib.GameInput{}
	count := 0
	playerNumByte := make([]byte, 8)
	for i := 0; i < len(playerNumByte); i++ {
		playerNumByte[i] = inputByte[count]
		count++
	}
	frameByte := make([]byte, 8)
	for i := 0; i < len(frameByte); i++ {
		frameByte[i] = inputByte[count]
		count++
	}
	bits := make([]byte, len(inputByte)-len(frameByte)-len(playerNumByte))
	for i := 0; i < len(bits); i++ {
		bits[i] = inputByte[count]
		count++
	}

	playerNum := int64(binary.LittleEndian.Uint64(playerNumByte))
	frame := int64(binary.LittleEndian.Uint64(frameByte))
	input.Frame = frame
	input.Bits = bits

	evt := Event{Input: input, PlayerNum: playerNum}
	return evt
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

func (n *Netplay) Disconnect() ggponet.GGPOErrorCode {
	n.Conn.Close()
	if n.Conn == nil {
		return ggponet.GGPO_OK
	}
	return ggponet.GGPO_ERRORCODE_PLAYER_DISCONNECTED
}
