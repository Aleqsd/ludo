package network

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"math/rand"
	"net"
	"unsafe"

	"github.com/libretro/ludo/ggpo/bitvector"
	"github.com/libretro/ludo/ggpo/ggponet"
	"github.com/libretro/ludo/ggpo/lib"
	"github.com/libretro/ludo/ggpo/platform"
	"github.com/sirupsen/logrus"
)

type Event struct {
	Input     lib.GameInput
	PlayerNum int64
}

type OoPacket struct {
	SendTime uint64
	DestAddr *net.UDPAddr
	Msg      *NetplayMsgType
}

type State int64

const (
	Syncing State = iota
	Synchronzied
	Running
	Disconnected
)

const (
	UDP_HEADER_SIZE           = 28 /* Size of IP + UDP headers */
	NUM_SYNC_PACKETS          = 5
	SYNC_RETRY_INTERVAL       = 2000
	SYNC_FIRST_RETRY_INTERVAL = 500
	RUNNING_RETRY_INTERVAL    = 200
	KEEP_ALIVE_INTERVAL       = 200
	QUALITY_REPORT_INTERVAL   = 1000
	NETWORK_STATS_INTERVAL    = 1000
	UDP_SHUTDOWN_TIMER        = 5000
	MAX_SEQ_DISTANCE          = (1 << 15)
)

type Netplay struct {
	Callbacks            ggponet.GGPOSessionCallbacks
	Poll                 lib.Poll
	Conn                 *net.UDPConn
	LocalAddr            *net.UDPAddr
	RemoteAddr           *net.UDPAddr
	Queue                int64
	IsHosting            bool
	LastReceivedInput    lib.GameInput
	LastAckedInput       lib.GameInput
	LastSentInput        lib.GameInput
	DisconnectEventSent  int64
	LocalConnectStatus   []ggponet.ConnectStatus
	LocalFrameAdvantage  int64
	RemoteFrameAdvantage int64
	RoundTripTime        int64
	KbpsSent             int64
	PeerConnectStatus    []ggponet.ConnectStatus
	TimeSync             lib.TimeSync
	CurrentState         State
	PendingOutput        lib.RingBuffer
	PacketsSent          int64
	BytesSent            int64
	LastSendTime         uint64
	MagicNumber          uint64
	RemoteMagicNumber    uint64
	Connected            bool
	NextSendSeq          uint64
	SendQueue            lib.RingBuffer
	SendLatency          int64
	OopPercent           int64
	OoPacket             OoPacket
	NetplayState         StateType
}

func (n *Netplay) Init(remotePlayer ggponet.GGPOPlayer, queue int64, status []ggponet.ConnectStatus, poll *lib.Poll /*, callbacks ggponet.GGPOSessionCallbacks*/) {
	//n.Callbacks = callbacks
	//n.Poll = poll
	//n.Poll.RegisterLoop(n)
	n.LocalAddr, _ = net.ResolveUDPAddr("udp4", "127.0.0.1:8089")
	n.RemoteAddr, _ = net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", remotePlayer.IPAddress, int(remotePlayer.Port)))
	n.Queue = queue
	n.LastReceivedInput.SimpleInit(-1, nil, 1)
	n.LastAckedInput.SimpleInit(-1, nil, 1)
	n.LastSentInput.SimpleInit(-1, nil, 1)
	n.LocalConnectStatus = status
	n.LocalFrameAdvantage = 0
	n.PendingOutput.Init(64)
	n.PacketsSent = 0
	n.BytesSent = 0
	n.LastSendTime = 0
	n.NextSendSeq = 0
	n.MagicNumber = 0
	n.SendQueue.Init(64)
	for n.MagicNumber == 0 {
		n.MagicNumber = rand.Uint64()
	}

	n.SendLatency = platform.GetConfigInt("ggpo.network.delay")
	n.OopPercent = platform.GetConfigInt("ggpo.oop.percent")
	n.OoPacket.Msg = nil

	var i lib.IPollSink = n
	poll.RegisterLoop(&i)

	logrus.Info(fmt.Sprintf("binding udp socket to port %d.", n.LocalAddr.Port))
}

func (n *Netplay) Write(msg *NetplayMsgType) {
	var err error
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	encoder.Encode(msg)
	if n.IsHosting {
		_, err = n.Conn.WriteToUDP(buffer.Bytes(), n.RemoteAddr)
	} else {
		_, err = n.Conn.Write(buffer.Bytes())
	}
	if err != nil {
		fmt.Println(err)
		return
	}
}

func (n *Netplay) Read() {
	var msg *NetplayMsgType
	for {
		netinput := make([]byte, 4096)
		length, _, err := n.Conn.ReadFromUDP(netinput)
		if err != nil {
			fmt.Println(err)
			//n.PeerConnectStatus = false
			return
		}
		//n.PeerConnectStatus = true
		buffer := bytes.NewBuffer(netinput[:length])
		decoder := gob.NewDecoder(buffer)
		decoder.Decode(msg)
		//TODO: Créer un channel pour stocker les inputs qui arrivent : channel <- msg
	}
}

func (n *Netplay) SendInput(input *lib.GameInput) {
	if n.CurrentState == Running {
		n.TimeSync.AdvanceFrame(input, n.LocalFrameAdvantage, n.RemoteFrameAdvantage)
		var t lib.T = &input
		n.PendingOutput.Push(&t)
	}
	n.SendPendingOutput()
}

func (n *Netplay) SendPendingOutput() {
	var msg *NetplayMsgType
	msg.Init(Input)
	offset := int64(0)
	var bits []byte
	var last lib.GameInput

	if n.PendingOutput.Size > 0 {
		last = n.LastAckedInput
		msg.Input.Bits = make([]byte, MAX_COMPRESSED_BITS)
		bits = msg.Input.Bits

		var input lib.GameInput = n.PendingOutput.Front().(lib.GameInput)
		msg.Input.StartFrame = input.Frame
		msg.Input.InputSize = input.Size

		for j := int64(0); j < n.PendingOutput.Size; j++ {
			current := n.PendingOutput.Item(j).(lib.GameInput)
			if bytes.Compare(current.Bits, last.Bits) != 0 {
				for i := int64(0); i < current.Size*8; i++ {
					if current.Value(i) != last.Value(i) {
						bitvector.SetBit(msg.Input.Bits, &offset)
						if current.Value(i) {
							bitvector.SetBit(bits, &offset)
						} else {
							bitvector.ClearBit(bits, &offset)
						}
						bitvector.WriteNibblet(bits, i, &offset)
					}
				}
			}
			bitvector.ClearBit(msg.Input.Bits, &offset)
			last = current
			n.LastSentInput = current
		}
	} else {
		msg.Input.StartFrame = 0
		msg.Input.InputSize = 0
	}
	msg.Input.AckFrame = n.LastReceivedInput.Frame
	msg.Input.NumBits = offset

	msg.Input.DisconnectRequested = n.CurrentState == Disconnected
	if n.LocalConnectStatus != nil {
		copy(msg.Input.PeerConnectStatus, n.LocalConnectStatus)
	} else {
		msg.Input.PeerConnectStatus = make([]ggponet.ConnectStatus, MSG_MAX_PLAYERS)
	}

	n.SendMsg(msg)
}

func (n *Netplay) SendMsg(msg *NetplayMsgType) {
	n.PacketsSent++
	n.LastSendTime = platform.GetCurrentTimeMS()
	n.BytesSent += msg.PacketSize()

	msg.Hdr.Magic = n.MagicNumber
	msg.Hdr.SequenceNumber = n.NextSendSeq
	n.NextSendSeq++

	var queue QueueEntry
	queue.Init(platform.GetCurrentTimeMS(), n.RemoteAddr, msg)
	var t lib.T = &queue
	n.SendQueue.Push(&t)
	n.PumpSendQueue()
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
	n.CurrentState = Disconnected
	n.Conn.Close()
	if n.Conn == nil {
		return ggponet.GGPO_OK
	}
	return ggponet.GGPO_ERRORCODE_PLAYER_DISCONNECTED
}

func (n *Netplay) GetPeerConnectStatus(id int64, frame *int64) bool {
	//TODO:
	return true
}

func (n *Netplay) OnInvalid(msg *MsgType, len int64) bool {
	return false
}

func (n *Netplay) OnSyncRequest(msg *NetplayMsgType, len int64) bool {
	if n.RemoteMagicNumber != 0 && msg.Hdr.Magic != n.RemoteMagicNumber {
		logrus.Info(fmt.Sprintf("Ignoring sync request from unknown endpoint (%d != %d).", msg.Hdr.Magic, n.RemoteMagicNumber))
		return false
	}
	var reply *NetplayMsgType
	reply.SyncReply.RandomReply = msg.SyncRequest.RandomRequest
	n.SendMsg(reply)
	return true
}

func (n *Netplay) OnSyncReply(msg *NetplayMsgType, len int64) bool {
	if n.CurrentState != Syncing {
		logrus.Info("Ignoring SyncReply while not synching.")
		return msg.Hdr.Magic == n.RemoteMagicNumber
	}

	if msg.SyncReply.RandomReply != int64(n.NetplayState.Sync.Random) {
		logrus.Info(fmt.Sprintf("sync reply %d != %d.  Keep looking...", msg.SyncReply.RandomReply, n.NetplayState.Sync.Random))
		return false
	}

	if !n.Connected {
		//QueueEvent(Event(Event.Connected))
		n.Connected = true
	}

	logrus.Info(fmt.Sprintf("Checking sync state (%d round trips remaining).", n.NetplayState.Sync.RoundTripsRemaining))
	n.NetplayState.Sync.RoundTripsRemaining--
	if n.NetplayState.Sync.RoundTripsRemaining == 0 {
		logrus.Info("Synchronized!")
		//TODO: Events
		//QueueEvent(Event(Event.Synchronized))
		n.CurrentState = Running
		n.LastReceivedInput.Frame = -1
		n.RemoteMagicNumber = msg.Hdr.Magic
	} else {
		//var evt Event = Event.Synchronizing
		//evt.u.Synchronizing.Total = NUM_SYNC_PACKETS
		//evt.u.Synchronizing.Count = NUM_SYNC_PACKETS - n.NetplayState.Sync.RoundTripsRemaining
		//QueueEvent(evt)
		//SendSyncRequest()
	}
	return true
}

func (n *Netplay) OnInput(msg *NetplayMsgType, len int64) bool {
	// If a disconnect is requested, go ahead and disconnect now.
	disconnectRequested := msg.Input.DisconnectRequested
	if disconnectRequested {
		if n.CurrentState != Disconnected && n.DisconnectEventSent == 0 {
			logrus.Info("Disconnecting endpoint on remote request.")
			//QueueEvent(Event(Event.Disconnected))
			n.DisconnectEventSent = 1
		}
	} else {
		// Update the peer connection status if this peer is still considered to be part of the network.
		remoteStatus := msg.Input.PeerConnectStatus
		for i := 0; i < int(unsafe.Sizeof(n.PeerConnectStatus)); i++ {
			if remoteStatus[i].LastFrame < n.PeerConnectStatus[i].LastFrame {
				logrus.Panic("Assert error remotestatus Lastframe")
				n.PeerConnectStatus[i].Disconnected = n.PeerConnectStatus[i].Disconnected || remoteStatus[i].Disconnected
				n.PeerConnectStatus[i].LastFrame = lib.MAX(n.PeerConnectStatus[i].LastFrame, remoteStatus[i].LastFrame)
			}
		}
	}

	// Decompress the input.

	lastReceivedFrameNumber := n.LastReceivedInput.Frame
	if msg.Input.NumBits > 0 {
		var offset int64 = 0
		bits := msg.Input.Bits
		numBits := msg.Input.NumBits
		currentFrame := msg.Input.StartFrame

		n.LastReceivedInput.Size = msg.Input.InputSize
		if n.LastReceivedInput.Frame < 0 {
			n.LastReceivedInput.Frame = msg.Input.StartFrame - 1
		}

		for offset < numBits {
			/*
			* Keep walking through the frames (parsing bits) until we reach
			* the inputs for the frame right after the one we're on.
			 */

			if currentFrame > n.LastReceivedInput.Frame+1 {
				logrus.Panic("Assert error currentframe")
			}
			var useInputs bool = currentFrame == n.LastReceivedInput.Frame+1

			for bitvector.ReadBit(bits, &offset) > 0 {
				on := bitvector.ReadBit(bits, &offset)
				button := bitvector.ReadNibblet(bits, &offset)
				if useInputs {
					if on > 0 {
						n.LastReceivedInput.Set(button)
					} else {
						n.LastReceivedInput.Clear(button)
					}
				}
			}
			if offset > numBits {
				logrus.Panic("Assert error offset > numBits")
			}

			// Now if we want to use these inputs, go ahead and send them to the emulator.
			if useInputs {
				// Move forward 1 frame in the stream.
				if currentFrame != n.LastReceivedInput.Frame+1 {
					logrus.Panic("Assert error currentFrame")
				}

				// Send the event to the emualtor
				// UdpProtocol::Event evt(UdpProtocol::Event::Input);
				// evt.u.input.input = _last_received_input;

				n.NetplayState.Running.LastInputPacketRecvTime = platform.GetCurrentTimeMS()
				logrus.Info(fmt.Sprintf("Sending frame %d to emu queue %d", n.LastReceivedInput.Frame, n.Queue))

				//QueueEvent(evt)
			} else {
				logrus.Info(fmt.Sprintf("Skipping past frame:(%d) current is %d.", currentFrame, n.LastReceivedInput.Frame))
			}

			// Move forward 1 frame in the input stream.
			currentFrame++
		}
	}
	if n.LastReceivedInput.Frame < lastReceivedFrameNumber {
		logrus.Panic("Assert error last received frame number")
	}

	// Get rid of our buffered input

	for n.PendingOutput.Size > 0 && (n.PendingOutput.Front()).(lib.GameInput).Frame < msg.Input.AckFrame {
		logrus.Info(fmt.Sprintf("Throwing away pending output frame %d\n", n.PendingOutput.Front().(lib.GameInput).Frame))
		n.LastAckedInput = n.PendingOutput.Front().(lib.GameInput)
		n.PendingOutput.Pop()
	}
	return true
}

func (n *Netplay) OnInputAck(msg *NetplayMsgType, len int64) bool {
	// Get rid of our buffered input
	for n.PendingOutput.Size > 0 && (n.PendingOutput.Front()).(lib.GameInput).Frame < msg.InputAck.AckFrame {
		logrus.Info(fmt.Sprintf("Throwing away pending output frame %d\n", n.PendingOutput.Front().(lib.GameInput).Frame))
		n.LastAckedInput = n.PendingOutput.Front().(lib.GameInput)
		n.PendingOutput.Pop()
	}
	return true
}

func (n *Netplay) OnQualityReport(msg *NetplayMsgType, len int64) bool {
	// send a reply so the other side can compute the round trip transmit time.
	//    UdpMsg *reply = new UdpMsg(UdpMsg::QualityReply);
	//    reply->u.quality_reply.pong = msg->u.quality_report.ping;
	//    SendMsg(reply);

	n.RemoteFrameAdvantage = msg.QualityReport.FrameAdvantage
	return true
}

func (n *Netplay) OnQualityReply(msg *NetplayMsgType, len int64) bool {
	n.RoundTripTime = int64(platform.GetCurrentTimeMS()) - msg.QualityReply.Pong
	return true
}

func (n *Netplay) OnKeepAlive(msg *NetplayMsgType, len int64) bool {
	return true
}

func (n *Netplay) GetNetworkStats(s *ggponet.GGPONetworkStats) {
	s.Network.Ping = n.RoundTripTime
	s.Network.SendQueueLen = n.PendingOutput.Size
	s.Network.KbpsSent = n.KbpsSent
	s.TimeSync.RemoteFramesBehind = n.RemoteFrameAdvantage
	s.TimeSync.LocalFramesBehind = n.LocalFrameAdvantage
}

func (n *Netplay) SetLocalFrameNumber(localFrame int64) {
	remoteFrame := n.LastReceivedInput.Frame + (n.RoundTripTime * 60 / 1000)
	n.LocalFrameAdvantage = remoteFrame - localFrame
}

func (n *Netplay) RecommendFrameDelay() int64 {
	return n.TimeSync.RecommendFrameWaitDuration(false)
}

func (n *Netplay) PumpSendQueue() {
	for !n.SendQueue.Empty() {
		var entry QueueEntry = n.SendQueue.Front().(QueueEntry)

		if n.SendLatency > 0 {
			// should really come up with a gaussian distributation based on the configured
			// value, but this will do for now.
			jitter := (n.SendLatency * 2 / 3) + ((rand.Int63() % n.SendLatency) / 3)
			if platform.GetCurrentTimeMS() < (n.SendQueue.Front().(QueueEntry).QueueTime + uint64(jitter)) {
				break
			}
		}
		if n.OopPercent > 0 && n.OoPacket.Msg == nil && ((rand.Int63() % 100) < n.OopPercent) {
			delay := rand.Int63() % (n.SendLatency*10 + 1000)
			logrus.Info(fmt.Sprintf("creating rogue oop (seq: %d  delay: %d)", entry.Msg.Hdr.SequenceNumber, delay))
			n.OoPacket.SendTime = platform.GetCurrentTimeMS() + uint64(delay)
			n.OoPacket.Msg = entry.Msg
			n.OoPacket.DestAddr = entry.DestAddr
		} else {
			n.Write(entry.Msg)
			entry.Msg = nil
		}
		n.SendQueue.Pop()
	}
	if n.OoPacket.Msg != nil && n.OoPacket.SendTime < platform.GetCurrentTimeMS() {
		logrus.Info("sending rogue oop!")
		n.Write(n.OoPacket.Msg)
		n.OoPacket.Msg = nil
	}
}

func (n *Netplay) OnLoopPoll() bool {
	return true
}
