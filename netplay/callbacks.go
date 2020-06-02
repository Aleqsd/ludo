package netplay

//TODO: Structure qui hérite de GGPOSessionCallbacks

import (
	"github.com/libretro/ludo/ggpo/ggponet"
)

type Callbacks struct{}

func (c *Callbacks) BeginGame(game string) bool {
	return true
}

func (c *Callbacks) SaveGameState(buffer **byte, len *int64, checksum *int64, frame int64) {

}

func (c *Callbacks) LoadGameState(buffer *byte, len int64) {

}

func (c *Callbacks) LogGameState(filename string, buffer *byte, len int64) {

}

func (c *Callbacks) FreeBuffer(buffer *byte) {

}

func (c *Callbacks) AdvanceFrame(flags int64) {

}

func (c *Callbacks) OnEvent(info *ggponet.GGPOEvent) {

}
