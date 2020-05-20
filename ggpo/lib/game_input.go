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
	if isize > GAMEINPUT_MAX_BYTES*GAMEINPUT_MAX_PLAYERS {
		return
	}
	g.Frame = iframe
	g.Size = isize
	g.Bits = make([]byte, GAMEINPUT_MAX_BYTES*GAMEINPUT_MAX_PLAYERS)
	if len(ibits) > 0 {
		for k := 0; k < int(offset*isize); k += int(isize) {
			for j := 0; j < int(isize); j++ {
				g.Bits[k+j] = ibits[j]
			}
		}
	}
}

func (g *GameInput) SimpleInit(iframe int64, ibits []byte, isize int64) {
	if isize > GAMEINPUT_MAX_BYTES*GAMEINPUT_MAX_PLAYERS {
		return
	}
	g.Frame = iframe
	g.Size = isize
	g.Bits = make([]byte, GAMEINPUT_MAX_BYTES*GAMEINPUT_MAX_PLAYERS)
	if len(ibits) > 0 {
		copy(g.Bits, ibits)
	}
}

func (g *GameInput) Equal(other GameInput, bitsonly bool) bool {
	return (bitsonly || g.Frame == other.Frame) && g.Size == other.Size && bytes.Compare(g.Bits, other.Bits) == 0
}

func (g *GameInput) Erase() {
	g.Bits = make([]byte, len(g.Bits))
}

func (g *GameInput) Value(i int64) bool {
	return (g.Bits[i/8] & (1 << (i % 8))) != 0
}
