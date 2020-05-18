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
	Spectators            [ggponet.GGPO_MAX_SPECTATORS]network.Netplay
	LocalConnectStatus    []ggponet.ConnectStatus
	Endpoints             []network.Netplay
	Players               []ggponet.GGPOPlayer
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

	p.Players = make([]ggponet.GGPOPlayer, p.NumPlayers)
	p.Endpoints = make([]network.Netplay, p.NumPlayers)
	p.LocalConnectStatus = make([]ggponet.ConnectStatus, p.NumPlayers)
	p.Sync.Init(config, p.LocalConnectStatus)
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
	p.Players[queue] = *player
	if player.PlayerNum < 1 || player.PlayerNum > p.NumPlayers {
		return ggponet.GGPO_ERRORCODE_PLAYER_OUT_OF_RANGE
	}
	*handle = p.QueueToPlayerHandle(queue)

	if player.Type == ggponet.GGPO_PLAYERTYPE_REMOTE {
		p.AddRemotePlayer(player, queue)
	}

	return ggponet.GGPO_OK
}

func (p *Peer2PeerBackend) AddRemotePlayer(player *ggponet.GGPOPlayer, queue int64) {
	p.Synchronizing = true
	p.Endpoints[queue].Init(*player, queue)
	if p.MustHostConnection(queue) {
		p.Endpoints[queue].HostConnection()
	} else {
		p.Endpoints[queue].JoinConnection()
	}
}

func (p *Peer2PeerBackend) MustHostConnection(other int64) bool {
	result := false
	for i := 0; i < int(other); i++ {
		if p.Players[i].Type == ggponet.GGPO_PLAYERTYPE_LOCAL {
			result = true
			break
		}
	}
	return result
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
		for i := 0; i < int(p.NumPlayers); i++ {
			p.Endpoints[i].SendInput(input)
		}
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

func (p *Peer2PeerBackend) DoPoll() ggponet.GGPOErrorCode {
	if !p.Sync.InRollback() {
		//_poll.Pump(0);

		p.PollNetplayEvents()

		if !p.Synchronizing {
			p.Sync.CheckSimulation()

			// notify all of our endpoints of their local frame number for their
			// next connection quality report
			currentFrame := p.Sync.FrameCount
			for i := 0; i < int(p.NumPlayers); i++ {
				p.Endpoints[i].SetLocalFrameNumber(currentFrame)
			}

			var totalMinConfirmed int64
			if p.NumPlayers <= 2 {
				totalMinConfirmed = p.Poll2Players(currentFrame)
			} else {
				totalMinConfirmed = p.PollNPlayers(currentFrame)
			}

			//Log("last confirmed frame in p2p backend is %d.\n", totalMinConfirmed);
			if totalMinConfirmed >= 0 {
				if p.NumSpectators > 0 {
					for p.NextSpectatorFrame <= totalMinConfirmed {
						//Log("pushing frame %d to spectators.\n", p.NextSpectatorFrame)

						var input lib.GameInput
						input.Frame = p.NextSpectatorFrame
						input.Size = p.InputSize * p.NumPlayers
						p.Sync.GetConfirmedInputs(input.Bits, p.InputSize*p.NumPlayers, p.NextSpectatorFrame)
						for i := 0; i < int(p.NumSpectators); i++ {
							p.Spectators[i].SendInput(input)
						}
						p.NextSpectatorFrame++
					}
				}
				//Log("setting confirmed frame in sync to %d.\n", totalMinConfirmed);
				p.Sync.SetLastConfirmedFrame(totalMinConfirmed)
			}

			// send timesync notifications if now is the proper time
			if currentFrame > p.NextRecommendedSleep {
				interval := int64(0)
				for i := 0; i < int(p.NumPlayers); i++ {
					interval = lib.MAX(interval, p.Endpoints[i].RecommendFrameDelay())
				}

				if interval > 0 {
					var info ggponet.GGPOEvent
					info.Code = ggponet.GGPO_EVENTCODE_TIMESYNC
					info.TimeSync.FramesAhead = interval
					p.Callbacks.OnEvent(&info)
					p.NextRecommendedSleep = currentFrame + RECOMMENDATION_INTERVAL
				}
			}
		}
	}
	return ggponet.GGPO_OK
}

func (p *Peer2PeerBackend) Poll2Players(currentFrame int64) int64 {
	totalMinConfirmed := int64(lib.MAX_INT)
	for i := 0; i < int(p.NumPlayers); i++ {
		queueConnected := p.Endpoints[i].PeerConnectStatus

		if p.LocalConnectStatus[i].Disconnected == 0 {
			totalMinConfirmed = lib.MIN(p.LocalConnectStatus[i].LastFrame, totalMinConfirmed)
		}
		//Log("  local endp: connected = %d, last_received = %d, totalMinConfirmed = %d.\n", !p.LocalConnectStatus[i].Disconnected, p.LocalConnectStatus[i].LastFrame, total_min_confirmed)
		if !queueConnected && p.LocalConnectStatus[i].Disconnected == 0 {
			//Log("disconnecting i %d by remote request.\n", i)
			p.DisconnectPlayerQueue(int64(i), totalMinConfirmed)
		}
		//Log("  totalMinConfirmed = %d.\n", totalMinConfirmed)
	}
	return totalMinConfirmed
}

func (p *Peer2PeerBackend) IncrementFrame() ggponet.GGPOErrorCode {
	p.Sync.IncrementFrame()
	p.DoPoll()
	//p.PollSyncEvents();

	return ggponet.GGPO_OK
}

func (p *Peer2PeerBackend) DisconnectPlayer(player ggponet.GGPOPlayerHandle) ggponet.GGPOErrorCode {
	var queue int64
	var result ggponet.GGPOErrorCode

	result = p.PlayerHandleToQueue(player, &queue)
	if !ggponet.GGPO_SUCCEEDED(result) {
		return result
	}

	if p.LocalConnectStatus[queue].Disconnected == 1 {
		return ggponet.GGPO_ERRORCODE_PLAYER_DISCONNECTED
	}

	currentFrame := p.Sync.GetFrameCount()
	//log.Logger("Disconnecting local player %d at frame %d by user request.\n", queue, p.LocalConnectStatus[queue].LastFrame)
	var i int64 = 0
	for ; i < p.NumPlayers; i++ {
		p.DisconnectPlayerQueue(i, currentFrame)
	}

	return ggponet.GGPO_OK
}

func (p *Peer2PeerBackend) DisconnectPlayerQueue(queue int64, syncto int64) {
	var info ggponet.GGPOEvent
	framecount := p.Sync.GetFrameCount()

	p.Endpoints[queue].Disconnect()

	p.LocalConnectStatus[queue].Disconnected = 1
	p.LocalConnectStatus[queue].LastFrame = syncto

	if syncto < framecount {
		p.Sync.AdjustSimulation(syncto)
	}

	info.Code = ggponet.GGPO_EVENTCODE_DISCONNECTED_FROM_PEER
	info.Disconnected.Player = p.QueueToPlayerHandle(queue)
	p.Callbacks.OnEvent(&info)
}

func (p *Peer2PeerBackend) QueueToPlayerHandle(queue int64) ggponet.GGPOPlayerHandle {
	return (ggponet.GGPOPlayerHandle)(queue + 1)
}

func (p *Peer2PeerBackend) GetNetworkStats(stats *ggponet.GGPONetworkStats, player ggponet.GGPOPlayerHandle) ggponet.GGPOErrorCode {
	var queue int64
	var result ggponet.GGPOErrorCode

	result = p.PlayerHandleToQueue(player, &queue)
	if !ggponet.GGPO_SUCCEEDED(result) {
		return result
	}

	//stats = make([]byte, unsafe.Sizeof(*stats))
	//p.Endpoints[queue].GetNetworkStats(stats)

	return ggponet.GGPO_OK
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
