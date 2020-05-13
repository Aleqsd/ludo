package lib

import "bytes"

const (
	GAMEINPUT_MAX_BYTES   = 9
	GAMEINPUT_MAX_PLAYERS = 2
	NULL_FRAME            = -1
)

type GameInput struct {
	Size  int64
	Frame int64
	Bits  []byte
}

func (g *GameInput) Init(iframe int64, ibits []byte, isize int64, offset int64) {

}

func (g *GameInput) SimpleInit(iframe int64, ibits []byte, isize int64) {
	if isize > GAMEINPUT_MAX_BYTES*GAMEINPUT_MAX_PLAYERS {
		return
	}
	g.Frame = iframe
	g.Size = isize
	g.Bits = []byte{}
	if len(ibits) > 0 {
		g.Bits = ibits
	}
}

func (g *GameInput) Equal(other GameInput, bitsonly bool) bool {
	return (bitsonly || g.Frame == other.Frame) && g.Size == other.Size && bytes.Compare(g.Bits, other.Bits) == 0
}

// input.go ?
