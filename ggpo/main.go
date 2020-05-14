package ggpo

import (
	"github.com/libretro/ludo/ggpo/backend"
	"github.com/libretro/ludo/ggpo/ggponet"
)

// StartSession begins our game session
func StartSession(session **ggponet.GGPOSession, cb ggponet.GGPOSessionCallbacks, game string, numPlayers int64, inputSize int64) ggponet.GGPOErrorCode {
	var p2p backend.Peer2PeerBackend = backend.Peer2PeerBackend{NumPlayers: numPlayers, InputSize: inputSize}
	p2p.Init(cb, game)
	var s ggponet.GGPOSession = &p2p
	*session = &s
	return ggponet.GGPO_OK
}

// AddPlayer allows to add player in our game session
func AddPlayer(ggpo *ggponet.GGPOSession, player *ggponet.GGPOPlayer, handle *ggponet.GGPOPlayerHandle) ggponet.GGPOErrorCode {
	if ggpo == nil {
		return ggponet.GGPO_ERRORCODE_INVALID_SESSION
	}
	return (*ggpo).AddPlayer(player, handle)
}

// SetFrameDelay is used to set frame delay to local inputs
func SetFrameDelay(ggpo *ggponet.GGPOSession, player ggponet.GGPOPlayerHandle, frameDelay int64) ggponet.GGPOErrorCode {
	if ggpo == nil {
		return ggponet.GGPO_ERRORCODE_INVALID_SESSION
	}
	return (*ggpo).SetFrameDelay(player, frameDelay)
}

// AddLocalInput is used to add a local input before fetching the inputs for the remote players
func AddLocalInput(ggpo *ggponet.GGPOSession, player ggponet.GGPOPlayerHandle, values []byte, size int64) ggponet.GGPOErrorCode {
	if ggpo == nil {
		return ggponet.GGPO_ERRORCODE_INVALID_SESSION
	}
	return (*ggpo).AddLocalInput(player, values, size)
}

// (Cette fonction n'a peut-être plus aucun sens dans la mesure où la réception des paquets va se faire en mode asynchrone)
// Idle is used to define the time we allow ggpo to spent receive packets from other players during 1 frame
func Idle(ggpo *ggponet.GGPOSession, timeout int64) ggponet.GGPOErrorCode {
	if ggpo == nil {
		return ggponet.GGPO_ERRORCODE_INVALID_SESSION
	}
	return (*ggpo).DoPoll(timeout)
}

func SynchronizeInput(ggpo *ggponet.GGPOSession, values []byte, size int64, disconnectFlags *int64) ggponet.GGPOErrorCode {
	if ggpo == nil {
		return ggponet.GGPO_ERRORCODE_INVALID_SESSION
	}
	return (*ggpo).SyncInput(values, size, disconnectFlags)
}
