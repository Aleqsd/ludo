package netplay

import (
	"github.com/libretro/ludo/ggpo"
	"github.com/libretro/ludo/ggpo/ggponet"
)

var Test = false
var ggpoSession *ggponet.GGPOSession = nil
var ngs NonGameState = NonGameState{}
var syncTest = false

const FRAME_DELAY = 2 //TODO: Make frame delay depends on local network connection

func Init(numPlayers int64, players []ggponet.GGPOPlayer, numSpectators int64, test bool) {
	var result ggponet.GGPOErrorCode
	syncTest = test

	// Initialize the game state
	//gs.Init(hwnd, num_players);
	ngs.NumPlayers = numPlayers

	// Fill in a ggpo callbacks structure to pass to start_session.
	var cb ggponet.GGPOSessionCallbacks = &Callbacks{}

	if syncTest {
		//result = ggpo.StartSynctest(&ggpoSession, &cb, "ludo", num_players, sizeof(int), 1)
	} else {
		//TODO: Define optimal input size (default 100)
		result = ggpo.StartSession(&ggpoSession, cb, "ludo", numPlayers, 100)
	}

	// automatically disconnect clients after 3000 ms and start our count-down timer
	// for disconnects after 1000 ms.   To completely disable disconnects, simply use
	// a value of 0 for ggpo_set_disconnect_timeout.
	ggpo.SetDisconnectTimeout(ggpoSession, 3000)
	ggpo.SetDisconnectNotifyStart(ggpoSession, 1000)

	for i := 0; i < int(numPlayers+numSpectators); i++ {
		var handle ggponet.GGPOPlayerHandle
		result = ggpo.AddPlayer(ggpoSession, &players[i], &handle)
		ngs.Players[i].Handle = handle
		ngs.Players[i].Type = players[i].Type
		if players[i].Type == ggponet.GGPO_PLAYERTYPE_LOCAL {
			ngs.Players[i].ConnectProgress = 100
			ngs.LocalPlayerHandle = handle
			ngs.SetConnectState(handle, Connecting)
			ggpo.SetFrameDelay(ggpoSession, handle, FRAME_DELAY)
		} else {
			ngs.Players[i].ConnectProgress = 0
		}
	}

	if result != ggponet.GGPO_OK {
		//TODO: panic
	}
}

func InitSpectator(numPlayers int64, hostIp string, hostPort uint8) {
	//TODO: Spectators
	//var result ggponet.GGPOErrorCode

	// Initialize the game state
	//gs.Init(hwnd, num_players);
	ngs.NumPlayers = numPlayers

	// Fill in a ggpo callbacks structure to pass to start_session.
	//var cb ggponet.GGPOSessionCallbacks = &Callbacks{}

	//result = ggpo_start_spectating(&ggpo, &cb, "vectorwar", num_players, sizeof(int), localport, host_ip, host_port)
}

func DisconnectPlayer(player int64) {
	if player < ngs.NumPlayers {
		var result ggponet.GGPOErrorCode = ggpo.DisconnectPlayer(ggpoSession, ngs.Players[player].Handle)
		if ggponet.GGPO_SUCCEEDED(result) {
			//sprintf_s(logbuf, ARRAYSIZE(logbuf), "Disconnected player %d.\n", player)
			//TODO: log
		} else {
			//sprintf_s(logbuf, ARRAYSIZE(logbuf), "Error while disconnecting player (err:%d).\n", result)
			//TODO: log
		}
	}
}

func AdvanceFrame(inputs []byte, disconnectFlags int64) {
	//gs.Update(inputs, disconnect_flags);

	// update the checksums to display in the top of the window.  this
	// helps to detect desyncs.
	//TODO: Handle gs
	/*ngs.Now.Framenumber = gs._framenumber;
	ngs.Now.Checksum = fletcher32_checksum((short *)&gs, sizeof(gs) / 2);
	if ((gs._framenumber % 90) == 0) {
		ngs.periodic = ngs.now;
	}*/

	// Notify ggpo that we've moved forward exactly 1 frame.
	ggpo.AdvanceFrame(ggpoSession)

	// Update the performance monitor display.
	var handles [MAX_PLAYERS]ggponet.GGPOPlayerHandle
	count := 0
	for i := 0; i < int(ngs.NumPlayers); i++ {
		if ngs.Players[i].Type == ggponet.GGPO_PLAYERTYPE_REMOTE {
			handles[count] = ngs.Players[i].Handle
			count++
		}
	}
}

//TODO: Define how to get inputs
func RunFrame() {
	var result ggponet.GGPOErrorCode = ggponet.GGPO_OK
	//var disconnectFlags int64
	//var inputs [MAX_SHIPS]int64 = { 0 };

	/*if ngs.LocalPlayerHandle != ggponet.GGPO_INVALID_HANDLE {
		int input = ReadInputs(hwnd)
		if syncTest {
			input = randInt64() // test: use random inputs to demonstrate sync testing
		}
		result = ggpo.AddLocalInput(ggpoSession, ngs.LocalPlayerHandle, &input, sizeof(input))
	}

	// synchronize these inputs with ggpo.  If we have enough input to proceed
	// ggpo will modify the input list with the correct inputs to use and
	// return 1.
	if ggponet.GGPO_SUCCEEDED(result) {
		result = ggpo.SynchronizeInput(ggpo, (void *)inputs, sizeof(int) * MAX_SHIPS, &disconnectFlags)
		if ggponet.GGPO_SUCCEEDED(result) {
			// inputs[0] and inputs[1] contain the inputs for p1 and p2.  Advance
			// the game by 1 frame using those inputs.
			ggpo.AdvanceFrame(inputs, disconnectFlags)
		}
	}*/

	if result != ggponet.GGPO_OK {
		//TODO: panic
	}
}

func Idle() {
	ggpo.Idle(ggpoSession)
}
