package lib

const (
	INPUT_QUEUE_LENGTH = 128
	DEFAULT_INPUT_SIZE = 4
)

func PREVIOUS_FRAME(offset int64) int64 {
	if offset == 0 {
		return INPUT_QUEUE_LENGTH - 1
	}
	return offset - 1
}

type InputQueue struct {
	ID                  int64
	Head                int64
	Tail                int64
	Length              int64
	FirstFrame          bool
	LastUserAddedFrame  int64
	LastAddedFrame      int64
	FirstIncorrectFrame int64
	LastFrameRequested  int64
	FrameDelay          int64
	Inputs              [INPUT_QUEUE_LENGTH]GameInput
	Prediction          GameInput
}

func (i *InputQueue) Init(id int64, inputSize int64) {
	i.ID = id
	i.Head = 0
	i.Tail = 0
	i.Length = 0
	i.FrameDelay = 0
	i.FirstFrame = true
	i.LastUserAddedFrame = NULL_FRAME
	i.FirstIncorrectFrame = NULL_FRAME
	i.LastFrameRequested = NULL_FRAME
	i.LastAddedFrame = NULL_FRAME
	i.Inputs = [INPUT_QUEUE_LENGTH]GameInput{}

	i.Prediction.SimpleInit(NULL_FRAME, nil, inputSize)

	for j := 0; j < len(i.Inputs); j++ {
		i.Inputs[j].Size = inputSize
	}
}

func (i *InputQueue) SetFrameDelay(delay int64) {
	i.FrameDelay = delay
}

func (i *InputQueue) DiscardConfirmedFrames(frame int64) {

}

func (i *InputQueue) ResetPrediction(frame int64) {

}

func (i *InputQueue) GetConfirmedInput(requestedFrame int64, input *GameInput) bool {
	return true
}

func (i *InputQueue) GetInput(requestedFrame int64, input *GameInput) bool {
	return true
}

func (i *InputQueue) AddInput(input GameInput) {
	var new_frame int64

	//Log("adding input frame number %d to queue.\n", input.frame);

	i.LastUserAddedFrame = input.Frame

	new_frame = i.AdvanceQueueHead(input.Frame)
	if new_frame != NULL_FRAME {
		i.AddDelayedInputToQueue(input, new_frame)
	}

	input.Frame = new_frame
}

func (i *InputQueue) AddDelayedInputToQueue(input GameInput, frameNumber int64) {
	//Log("adding delayed input frame number %d to queue.\n", frame_number)
	i.Inputs[i.Head] = input
	i.Inputs[i.Head].Frame = frameNumber
	i.Head = (i.Head + 1) % INPUT_QUEUE_LENGTH
	i.Length++
	i.FirstFrame = false

	i.LastAddedFrame = frameNumber

	if i.Prediction.Frame != NULL_FRAME {
		if i.FirstIncorrectFrame == NULL_FRAME && !i.Prediction.Equal(input, true) {
			//Log("frame %d does not match prediction.  marking error.\n", frameNumber)
			i.FirstIncorrectFrame = frameNumber
		}

		if i.Prediction.Frame == i.LastFrameRequested && i.FirstIncorrectFrame == NULL_FRAME {
			//Log("prediction is correct!  dumping out of prediction mode.\n")
			i.Prediction.Frame = NULL_FRAME
		} else {
			i.Prediction.Frame++
		}
	}
}

func (i *InputQueue) AdvanceQueueHead(frame int64) int64 {
	//Log("advancing queue head to frame %d.\n", frame)

	expectedFrame := i.Inputs[PREVIOUS_FRAME(i.Head)].Frame + 1
	if i.FirstFrame {
		expectedFrame = 0
	}

	frame += i.FrameDelay

	if expectedFrame > frame {
		//Log("Dropping input frame %d (expected next frame to be %d).\n", frame, expected_frame);
		return NULL_FRAME
	}

	for expectedFrame < frame {
		//Log("Adding padding frame %d to account for change in frame delay.\n", expected_frame)
		var lastFrame GameInput = i.Inputs[PREVIOUS_FRAME(i.Head)]
		i.AddDelayedInputToQueue(lastFrame, expectedFrame)
		expectedFrame++
	}

	return frame
}
