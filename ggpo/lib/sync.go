package lib

import (
	"unsafe"

	"github.com/libretro/ludo/ggpo/ggponet"
)

const MAX_PREDICTION_FRAMES = 8

type Sync struct {
	Rollingback         bool
	LastConfirmedFrame  int64
	FrameCount          int64
	MaxPredictionFrames int64
	SavedState          SavedState
	InputQueues         []InputQueue
	Config              Config
	Callbacks           ggponet.GGPOSessionCallbacks
	LocalConnectStatus  []ggponet.ConnectStatus
}

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

func (s *Sync) Init(config Config, ConnectStatus []ggponet.ConnectStatus) {
	s.Config = config
	s.Callbacks = config.Callbacks
	s.FrameCount = 0
	s.Rollingback = false
	s.LocalConnectStatus = ConnectStatus

	s.MaxPredictionFrames = config.NumPredictionFrames

	s.CreateQueues(config)
}

func (s *Sync) SetLastConfirmedFrame(frame int64) {   
   s.LastConfirmedFrame = frame
   if s.LastConfirmedFrame > 0 {
      for i := 0; i < int(s.Config.NumPlayers); i++ {
         s.InputQueues[i].DiscardConfirmedFrames(frame - 1)
      }
   }
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

func (s *Sync) GetConfirmedInputs(values []byte, size int64, frame int64) int64 {
	disconnectFlags := 0
	output := values

	for i := 0; i < int(s.Config.NumPlayers); i++ {
		var input GameInput
		if s.LocalConnectStatus[i].Disconnected == 1 && frame > s.LocalConnectStatus[i].LastFrame {
			disconnectFlags |= (1 << i)
			input.Erase()
		} else {
			s.InputQueues[i].GetConfirmedInput(frame, &input)
		}
		for k := 0; k < i*int(s.Config.InputSize); k += int(s.Config.InputSize) {
			for j := 0; j < int(s.Config.InputSize); j++ {
				output[k+j] = input.Bits[j]
			}
		}
	}
	return int64(disconnectFlags)
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

func (s *Sync) SynchronizeInputs(values []byte, size int64) int64 {
	var disconnectedFlags int64 = 0
	output := values

	output = make([]byte, size)
	for i := 0; i < int(s.Config.NumPlayers); i++ {
		var input GameInput
		if s.LocalConnectStatus[i].Disconnected == 1 && s.FrameCount > s.LocalConnectStatus[i].LastFrame {
			disconnectedFlags += 1 << i
			input.Bits = nil
		} else {
			s.InputQueues[i].GetInput(s.FrameCount, &input)
		}
		output = make([]byte, len(input.Bits)*i+len(output))
		output = input.Bits
	}
	return disconnectedFlags
}

func (s *Sync) CheckSimulation() {
	var seek_to int64
	if !s.CheckSimulationConsistency(&seek_to) {
		s.AdjustSimulation(seek_to)
	}
}

func (s *Sync) IncrementFrame() {
	s.FrameCount++
	s.SaveCurrentFrame()
}

func (s *Sync) CheckSimulationConsistency(seekTo *int64) bool {
	firstIncorrect := NULL_FRAME
	for i := 0; i < int(s.Config.NumPlayers); i++ {
		incorrect := s.InputQueues[i].FirstIncorrectFrame
		//Log("considering incorrect frame %d reported by queue %d.\n", incorrect, i)

		if incorrect != NULL_FRAME && (firstIncorrect == NULL_FRAME || incorrect < int64(firstIncorrect)) {
			firstIncorrect = int(incorrect)
		}
	}

	if firstIncorrect == NULL_FRAME {
		//Log("prediction ok.  proceeding.\n")
		return true
	}
	*seekTo = int64(firstIncorrect)
	return false
}

func (s *Sync) AdjustSimulation(seekTo int64) {
	count := s.FrameCount - seekTo

	s.Rollingback = true

	// Flush our input queue and load the last frame
	s.LoadFrame(seekTo)

	//Advance frame by frame (stuffign notifications back to master)
	s.ResetPrediction(s.FrameCount)
	for i := 0; i < int(count); i++ {
		s.Callbacks.AdvanceFrame(0)
	}

	s.Rollingback = false
}

func (s *Sync) LoadFrame(frame int64) {

	// Find the frame in question
	if frame == s.FrameCount {
		return
	}

	// Move the head pointer back and load it up
	s.SavedState.head = s.FindSavedFrameIndex(frame)

	var state *SavedFrame = &s.SavedState.frames[s.SavedState.head]

	//Log("=== Loading frame info %d (size: %d  checksum: %08x).\n",state->frame, state->cbuf, state->checksum);

	s.Callbacks.LoadGameState(state.buf, state.cbuf)

	// Reset framecount and the head of the state ring-buffer to point in
	// advance of the current frame (as if we had just finished executing it).

	s.FrameCount = state.frame
	s.SavedState.head = s.SavedState.head + 1%int64(unsafe.Sizeof(s.SavedState.frames))
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

func (s *Sync) FindSavedFrameIndex(frame int64) int64 {
	var i int64 = int64(unsafe.Sizeof(s.SavedState.frames))
	var count int64 = int64(unsafe.Sizeof(s.SavedState.frames))
	for i = 0; i < count; i++ {
		if s.SavedState.frames[i].frame == frame {
			break
		}
	}
	if i == count {
		panic("FindSavedFrameIndex i = count")
	}
	return i
}

func (s *Sync) CreateQueues(config Config) bool {
	s.InputQueues = make([]InputQueue, config.NumPlayers)

	for i := 0; i < int(s.Config.NumPlayers); i++ {
		s.InputQueues[i].Init(int64(i), s.Config.InputSize)
	}
	return true
}

func (s *Sync) ResetPrediction(frameNumber int64) {
	for i := 0; i < int(s.Config.NumPlayers); i++ {
		s.InputQueues[i].ResetPrediction(frameNumber)
	}
}
