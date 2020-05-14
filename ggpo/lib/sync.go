package lib

import "github.com/libretro/ludo/ggpo/ggponet"

const MAX_PREDICTION_FRAMES = 8

type SavedFrame struct {
	buf      *byte
	cbuf     int64
	frame    int64
	checksum int64
}

func (s *SavedFrame) Init() {
	s.buf = nil
	s.cbuf = 0
	s.frame = -1
	s.checksum = 0
}

type SavedState struct {
	frames [MAX_PREDICTION_FRAMES + 2]SavedFrame
	head   int64
}

type Config struct {
	Callbacks           ggponet.GGPOSessionCallbacks
	NumPredictionFrames int64
	NumPlayers          int64
	InputSize           int64
}

type Event struct {
	ConfirmedInput int64
	Input          GameInput
}

type Sync struct {
	Rollingback         bool
	LastConfirmedFrame  int64
	FrameCount          int64
	MaxPredictionFrames int64
	SavedState          SavedState
	InputQueues         []InputQueue
	Config              Config
	Callbacks           ggponet.GGPOSessionCallbacks
}

func (s *Sync) Init(config Config) {
	s.Config = config
	s.Callbacks = config.Callbacks
	s.FrameCount = 0
	s.Rollingback = false

	s.MaxPredictionFrames = config.NumPredictionFrames

	s.CreateQueues(config)
}

func (s *Sync) SetFrameDelay(queue int64, delay int64) {
	s.InputQueues[queue].SetFrameDelay(delay)
}

func (s *Sync) InRollback() bool {
	return s.Rollingback
}

func (s *Sync) GetFrameCount() int64 {
	return s.FrameCount
}

func (s *Sync) AddLocalInput(queue int64, input *GameInput) bool {
	framesBehind := s.FrameCount - s.LastConfirmedFrame
	if s.FrameCount >= s.MaxPredictionFrames && framesBehind >= s.MaxPredictionFrames {
		//Log("Rejecting input from emulator: reached prediction barrier.\n")
		return false
	}

	if s.FrameCount == 0 {
		s.SaveCurrentFrame()
	}

	//Log("Sending undelayed local frame %d to queue %d.\n", s.FrameCount, queue)
	input.Frame = s.FrameCount
	s.InputQueues[queue].AddInput(input)

	return true
}

// SaveCurrentFrame write everything into the head, then advance the head pointer
func (s *Sync) SaveCurrentFrame() {
	var state *SavedFrame = &s.SavedState.frames[s.SavedState.head]
	if state.buf != nil {
		s.Callbacks.FreeBuffer(state.buf)
		state.buf = nil
	}
	state.frame = s.FrameCount
	s.Callbacks.SaveGameState(&state.buf, &state.cbuf, &state.checksum, state.frame)

	//Log("=== Saved frame info %d (size: %d  checksum: %08x).\n", state->frame, state->cbuf, state->checksum)
	s.SavedState.head = (s.SavedState.head + 1) % int64(len(s.SavedState.frames))
}

func (s *Sync) CreateQueues(config Config) bool {
	s.InputQueues = make([]InputQueue, config.NumPlayers)

	for i := 0; i < int(s.Config.NumPlayers); i++ {
		s.InputQueues[i].Init(int64(i), s.Config.InputSize)
	}
	return true
}
