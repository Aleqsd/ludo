package netplay

import (
	"github.com/libretro/ludo/ggpo"
	"github.com/libretro/ludo/ggpo/ggponet"
)

var Test = false
var ggpoSession *ggponet.GGPOSession = nil
const FRAME_DELAY = 2 //TODO: Make frame delay depends on local network connection

//TODO: Non-gamestate?

func Init(numPlayers int64, players []ggponet.GGPOPlayer, numSpectators int64, test bool) {
	var result ggponet.GGPOErrorCode

	// Initialize the game state
	//gs.Init(hwnd, num_players);
	//ngs.num_players = num_players;

	// Fill in a ggpo callbacks structure to pass to start_session.
	var cb ggponet.GGPOSessionCallbacks = &Callbacks{}

	if test {
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
		//ngs.players[i].Handle = Handle
		//ngs.players[i].Type = players[i].Type
		if players[i].Type == ggponet.GGPO_PLAYERTYPE_LOCAL {
			//ngs.players[i].connect_progress = 100
			//ngs.local_player_handle = handle
			//ngs.SetConnectState(handle, Connecting)
			ggpo.SetFrameDelay(ggpoSession, handle, FRAME_DELAY)
		} else {
			//ngs.players[i].connect_progress = 0
		}
	}
}
