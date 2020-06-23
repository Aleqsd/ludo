package netplay

//TODO: Structure qui hérite de GGPOSessionCallbacks

import (
	"time"

	"github.com/libretro/ludo/ggpo"
	"github.com/libretro/ludo/ggpo/ggponet"
	local "github.com/libretro/ludo/input"
)

func GetCurrentTimeMS() uint64 {
	return uint64(time.Now().UnixNano() / int64(time.Millisecond))
}

var Synchronized = false

type Callbacks struct{}

func (c *Callbacks) BeginGame(game string) bool {
	return true
}

func (c *Callbacks) SaveGameState(buffer **byte, len *int64, checksum *int64, frame int64) {
	/* *len = sizeof(gs);
	*buffer = (unsigned char *)malloc(*len);
	if (!*buffer) {
		return false;
	}
	memcpy(*buffer, &gs, *len);
	*checksum = fletcher32_checksum((short *)*buffer, *len / 2);*/
}

func (c *Callbacks) LoadGameState(buffer *byte, len int64) {
	//memcpy(&gs, buffer, len)
}

func (c *Callbacks) LogGameState(filename string, buffer *byte, len int64) {

}

func (c *Callbacks) FreeBuffer(buffer *byte) {
	//TODO: Remove 'cause useless
}

func (c *Callbacks) AdvanceFrame(flags int64) {
	inputs := make([]byte, int64(local.ActionLast*ggponet.GGPO_MAX_PLAYERS))
	var disconnectFlags int64

	// Make sure we fetch new inputs from GGPO and use those to update
	// the game state instead of reading from the keyboard.
	ggpo.SynchronizeInput(ggpoSession, inputs, int64(local.ActionLast*ggponet.GGPO_MAX_PLAYERS), &disconnectFlags)
	AdvanceFrame(inputs, disconnectFlags)
}

func (c *Callbacks) OnEvent(info *ggponet.GGPOEvent) {
	var progress int64
	switch info.Code {
	case ggponet.GGPO_EVENTCODE_CONNECTED_TO_PEER:
		ngs.SetConnectState(info.Connected.Player, Synchronizing)
		break
	case ggponet.GGPO_EVENTCODE_SYNCHRONIZING_WITH_PEER:
		progress = 100 * info.Synchronizing.Count / info.Synchronizing.Total
		ngs.UpdateConnectProgress(info.Synchronizing.Player, progress)
		break
	case ggponet.GGPO_EVENTCODE_SYNCHRONIZED_WITH_PEER:
		ngs.UpdateConnectProgress(info.Synchronized.Player, 100)
		Synchronized = true
		break
	case ggponet.GGPO_EVENTCODE_RUNNING:
		ngs.SetAllConnectState(Running)
		break
	case ggponet.GGPO_EVENTCODE_CONNECTION_INTERRUPTED:
		ngs.SetDisconnectTimeout(info.ConnectionInterrupted.Player, int64(GetCurrentTimeMS()), info.ConnectionInterrupted.DisconnectTimeout)
		break
	case ggponet.GGPO_EVENTCODE_CONNECTION_RESUMED:
		ngs.SetConnectState(info.ConnectionResumed.Player, Running)
		break
	case ggponet.GGPO_EVENTCODE_DISCONNECTED_FROM_PEER:
		ngs.SetConnectState(info.Disconnected.Player, Disconnected)
		break
	case ggponet.GGPO_EVENTCODE_TIMESYNC:
		time.Sleep((time.Duration)(1000 * info.TimeSync.FramesAhead / 60))
		break
	}
}
