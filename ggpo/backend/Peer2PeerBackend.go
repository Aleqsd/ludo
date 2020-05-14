package backend

import (
	"github.com/libretro/ludo/ggpo/ggponet"
	"github.com/libretro/ludo/ggpo/lib"
	"github.com/libretro/ludo/ggpo/network"
)

const (
	RECOMMENDATION_INTERVAL         = 240
	DEFAULT_DISCONNECT_TIMEOUT      = 5000
	DEFAULT_DISCONNECT_NOTIFY_START = 750
)

type Peer2PeerBackend struct {
	//Poll                  _poll;
	//UdpProtocol           _spectators[ggponet.GGPO_MAX_SPECTATORS];
	LocalConnectStatus    []ggponet.ConnectStatus
	Netplay               network.Netplay
	Endpoints             []ggponet.GGPOPlayer
	Sync                  lib.Sync
	InputSize             int64
	NumPlayers            int64
	NumSpectators         int64
	NextSpectatorFrame    int64
	NextRecommendedSleep  int64
	DisconnectTimeout     int64
	DisconnectNotifyStart int64
	Synchronizing         bool
	Callbacks             ggponet.GGPOSessionCallbacks
}

func (p *Peer2PeerBackend) Init(cb ggponet.GGPOSessionCallbacks, gamename string) {
	p.Callbacks = cb
	p.Synchronizing = true
	p.NextRecommendedSleep = 0
	var config lib.Config = lib.Config{}
	config.NumPlayers = p.NumPlayers
	config.InputSize = p.InputSize
	config.Callbacks = p.Callbacks
	config.NumPredictionFrames = lib.MAX_PREDICTION_FRAMES
	p.Sync.Init(config, p.LocalConnectStatus)

	p.Endpoints = make([]ggponet.GGPOPlayer, p.NumPlayers)
	p.LocalConnectStatus = make([]ggponet.ConnectStatus, p.NumPlayers)
	for i := 0; i < len(p.LocalConnectStatus); i++ {
		p.LocalConnectStatus[i].LastFrame = -1
	}

	p.Callbacks.BeginGame(gamename)
}

func (p *Peer2PeerBackend) AddPlayer(player *ggponet.GGPOPlayer, handle *ggponet.GGPOPlayerHandle) ggponet.GGPOErrorCode {
	if player.Type == ggponet.GGPO_PLAYERTYPE_SPECTATOR {
		return p.AddSpectator(player.IPAddress, player.Port)
	}

	queue := player.PlayerNum - 1
	p.Endpoints[queue] = *player
	if player.PlayerNum < 1 || player.PlayerNum > p.NumPlayers {
		return ggponet.GGPO_ERRORCODE_PLAYER_OUT_OF_RANGE
	}
	*handle = p.QueueToPlayerHandle(queue)

	if player.Type == ggponet.GGPO_PLAYERTYPE_LOCAL {
		return p.JoinRemotePlayer(queue)
	}

	return ggponet.GGPO_OK
}

func (p *Peer2PeerBackend) JoinRemotePlayer(queue int64) ggponet.GGPOErrorCode {
	p.Netplay.Init(p.Endpoints[queue], p.Endpoints[0])
	if queue == 0 {
		if !p.Netplay.HostConnection() {
			return ggponet.GGPO_ERRORCODE_PLAYER_DISCONNECTED
		}
	} else {
		if !p.Netplay.JoinConnection() {
			return ggponet.GGPO_ERRORCODE_PLAYER_DISCONNECTED
		}
	}
	p.Synchronizing = true

	return ggponet.GGPO_OK
}

func (p *Peer2PeerBackend) AddLocalInput(player ggponet.GGPOPlayerHandle, values []byte, size int64) ggponet.GGPOErrorCode {
	var queue int64
	var input lib.GameInput
	var result ggponet.GGPOErrorCode

	if p.Sync.InRollback() {
		return ggponet.GGPO_ERRORCODE_IN_ROLLBACK
	}
	if p.Synchronizing {
		return ggponet.GGPO_ERRORCODE_NOT_SYNCHRONIZED
	}

	result = p.PlayerHandleToQueue(player, &queue)
	if !ggponet.GGPO_SUCCEEDED(result) {
		return result
	}

	input.SimpleInit(-1, values, size)

	// Feed the input for the current frame into the synchronzation layer.
	if !p.Sync.AddLocalInput(queue, &input) {
		return ggponet.GGPO_ERRORCODE_PREDICTION_THRESHOLD
	}

	if input.Frame != lib.NULL_FRAME {
		//Log("setting local connect status for local queue %d to %d", queue, input.frame);
		p.LocalConnectStatus[queue].LastFrame = input.Frame

		// Send the input to all the remote players.
		p.Netplay.SendInput(input)
	}

	return ggponet.GGPO_OK
}

func (p *Peer2PeerBackend) SyncInput(values []byte, size int64, disconnectFlags *int64) ggponet.GGPOErrorCode {
	var flags int64

	// Wait until we've started to return inputs
	if p.Synchronizing {
		return ggponet.GGPO_ERRORCODE_NOT_SYNCHRONIZED
	}

	flags = p.Sync.SynchronizeInputs(values, size)
	if *disconnectFlags != 0 {
		*disconnectFlags = flags
	}

	return ggponet.GGPO_OK
}

func (p *Peer2PeerBackend) DoPoll(timeout int64) ggponet.GGPOErrorCode {
	return ggponet.GGPO_OK
}

func (p *Peer2PeerBackend) IncrementFrame(value byte) ggponet.GGPOErrorCode {
	return ggponet.GGPO_OK
}

func (p *Peer2PeerBackend) DisconnectPlayer(player ggponet.GGPOPlayerHandle) ggponet.GGPOErrorCode {
	return ggponet.GGPO_OK
}

func (p *Peer2PeerBackend) QueueToPlayerHandle(queue int64) ggponet.GGPOPlayerHandle {
	return (ggponet.GGPOPlayerHandle)(queue + 1)
}

func (p *Peer2PeerBackend) SetFrameDelay(player ggponet.GGPOPlayerHandle, delay int64) ggponet.GGPOErrorCode {
	var queue int64
	var result ggponet.GGPOErrorCode

	result = p.PlayerHandleToQueue(player, &queue)
	if !ggponet.GGPO_SUCCEEDED(result) {
		return result
	}
	p.Sync.SetFrameDelay(queue, delay)

	return ggponet.GGPO_OK
}

func (p *Peer2PeerBackend) PlayerHandleToQueue(player ggponet.GGPOPlayerHandle, queue *int64) ggponet.GGPOErrorCode {
	offset := ((int64)(player) - 1)
	if offset < 0 || offset >= p.NumPlayers {
		return ggponet.GGPO_ERRORCODE_INVALID_PLAYER_HANDLE
	}
	*queue = offset
	return ggponet.GGPO_OK
}

func (p *Peer2PeerBackend) AddSpectator(ip string, port uint8) ggponet.GGPOErrorCode {
	//TODO: Spectators
	return ggponet.GGPO_OK
}
