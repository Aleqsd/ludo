package backend

import (
	"github.com/libretro/ludo/ggpo/ggponet"
	"github.com/libretro/ludo/ggpo/lib"
)

type SyncTestBackend struct {
	Callbacks     ggponet.GGPOSessionCallbacks
	NumPlayers    int64
	CheckDistance int64
	LastVerified  int64
	RollingBack   bool
	Running       bool
	Logfp         string
	Game          string
	CurrentInput  lib.GameInput
	LastInput     lib.GameInput
	SavedFrame    [32]SavedInfo
	Sync          lib.Sync
}

type SavedInfo struct {
	frame    int64
	checksum int64
	buf      string
	cbuf     int64
	input    lib.GameInput
}

func (s *SyncTestBackend) Init(cb *ggponet.GGPOSessionCallbacks, gamename string, frames int64, numPlayers int64) {
	s.Callbacks = *cb
	s.NumPlayers = numPlayers
	s.CheckDistance = frames
	s.LastVerified = 0
	s.RollingBack = false
	s.Running = false
	s.Logfp = ""
	s.Game = gamename
	s.CurrentInput.Erase()

	var config lib.Config
	config.Callbacks = s.Callbacks
	config.NumPredictionFrames = lib.MAX_PREDICTION_FRAMES
	//TODO SyncInit ??
	// lib.Sync.Init(config)

	s.Callbacks.BeginGame(s.Game)
}

func (s *SyncTestBackend) DoPoll(timeout int64) ggponet.GGPOErrorCode {
	if !s.Running {
		var info ggponet.GGPOEvent

		info.Code = ggponet.GGPO_EVENTCODE_RUNNING
		s.Callbacks.OnEvent(&info)
		s.Running = true
	}
	return ggponet.GGPO_OK
}

func (s *SyncTestBackend) AddPlayer(player *ggponet.GGPOPlayer, handle *ggponet.GGPOPlayerHandle) ggponet.GGPOErrorCode {
	if player.PlayerNum < 1 || player.PlayerNum > s.NumPlayers {
		return ggponet.GGPO_ERRORCODE_PLAYER_OUT_OF_RANGE
	}
	//TODO Conversion handle to int ?
	//*handle = player.PlayerNum-1
	return ggponet.GGPO_OK
}

func (s *SyncTestBackend) AddLocalInput(player ggponet.GGPOPlayerHandle, values []byte, size int64) ggponet.GGPOErrorCode {
	if !s.Running {
		return ggponet.GGPO_ERRORCODE_NOT_SYNCHRONIZED
	}

	var index int64 = int64(player)
	for i := 0; i < int(size); i++ {
		s.CurrentInput.Bits[(index * size)] += values[i]
	}
	return ggponet.GGPO_OK
}

func (s *SyncTestBackend) SyncInput(values []byte, size int64, disconnectFlags *int64) {
	s.BeginLog(false)
	if s.RollingBack {
		s.LastInput = s.SavedFrame[0].input // front() of Ringbuffer = 0 ?
	} else {
		if s.Sync.GetFrameCount() == 0 {
			s.Sync.SaveCurrentFrame()
		}
		s.LastInput = s.CurrentInput
	}
	s.LastInput.Bits = values
	//TODO if *int ?
	//if disconnectFlags {
	//	*disconnectFlags = 0
	//}
}

func (s *SyncTestBackend) IncrementFrame() ggponet.GGPOErrorCode {
	s.Sync.IncrementFrame()
	s.CurrentInput.Erase()

	//    Log("End of frame(%d)...\n", _sync.GetFrameCount());
	s.Endlog()

	if s.RollingBack {
		return ggponet.GGPO_OK
	}

	frame := s.Sync.GetFrameCount()
	// Hold onto the current frame in our queue of saved states.  We'll need
	// the checksum later to verify that our replay of the same frame got the
	// same results.
	var info SavedInfo
	info.frame = frame
	info.input = s.LastInput
	//TODO no field cbuf in savedFrame ??
	//info.cbuf = s.Sync.GetLastSavedFrame().cbuf
	// info.buf = (char *)malloc(info.cbuf);
	// memcpy(info.buf, _sync.GetLastSavedFrame().buf, info.cbuf);
	// info.checksum = _sync.GetLastSavedFrame().checksum;
	// _saved_frames.push(info);

	if frame-s.LastVerified == s.CheckDistance {
		// We've gone far enough ahead and should now start replaying frames.
		// Load the last verified frame and set the rollback flag to true.
		s.Sync.LoadFrame(s.LastVerified)
		s.RollingBack = true
		for !s.SavedFrame.Empty() {
			s.Callbacks.AdvanceFrame(0)

			// Verify that the checksumn of this frame is the same as the one in our list
		}
	}
}

func (s *SyncTestBackend) RaiseSyncError() {

}

func (s *SyncTestBackend) Logv() {

}

func (s *SyncTestBackend) BeginLog(saving bool) {

}

func (s *SyncTestBackend) Endlog() {

}

func (s *SyncTestBackend) LogSaveStates() {

}
