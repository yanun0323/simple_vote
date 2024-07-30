package room

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"main/internal/utils"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/yanun0323/pkg/logs"
)

const (
	_countdownDuration = 5 * time.Second

	// Time allowed to write a message to the peer.
	_writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	_pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	_pingPeriod = (_pongWait * 9) / 10

	// Maximum message size allowed from peer.
	_maxMessageSize = 512
)

type HostWsMessageIncoming struct {
	Connect bool                          `json:"connect"`
	SetGame *HostWsMessageSetGameIncoming `json:"set_game"`
	Round   *HostWsMessageRoundIncoming   `json:"round"`
}

type (
	HostWsMessageSetGameIncoming struct {
		Candidates []*Candidate `json:"candidates"`
	}
	HostWsMessageRoundIncoming struct {
		Round    int  `json:"round"`
		Start    bool `json:"start"`
		GameOver bool `json:"game_over"`
	}
)

type HostWsMessageOutgoing struct {
	Connect   *HostWsMessageConnectResponse   `json:"connect,omitempty"`
	Round     *HostWsMessageRoundResponse     `json:"round,omitempty"`
	Dashboard *HostWsMessageDashboardResponse `json:"dashboard,omitempty"`
	Timestamp int64                           `json:"timestamp"`
}

type (
	HostWsMessageConnectResponse struct {
		Dashboard []*Candidate `json:"dashboard"`
		Round     int          `json:"round"`
		EndTime   int64        `json:"end_time"`
		GameOver  bool         `json:"game_over"`
	}

	HostWsMessageRoundResponse struct {
		Round    int   `json:"round"`
		EndTime  int64 `json:"end_time"`
		GameOver bool  `json:"game_over"`
	}

	HostWsMessageDashboardResponse struct {
		Dashboard []*Candidate `json:"dashboard"`
		GameOver  bool         `json:"game_over"`
	}
)

func ConnectHost() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logs.New(logs.LevelDebug).WithField("host", r.RequestURI)
		l.Info("wss request received")
		utils.SetWss(r)

		roomID, uid := r.PathValue("room_id"), r.PathValue("uid")
		if roomID == "" || uid == "" {
			l.Warn("roomID or uid is empty")
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		l = logs.New(logs.LevelDebug).WithField("host", uid)

		if roomID != uid {
			l.Warn("roomID and uid are not equal")
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		room, ok := _roomPool.Load(roomID)
		if !ok {
			l.Warn("room not found")
			w.WriteHeader(http.StatusNotFound)

			return
		}

		conn, err := _upgrade.Upgrade(w, r, nil)
		if err != nil {
			l.Errorf("upgrade, err: %+v", err)
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		h := Host{
			l:   l,
			UID: uid,
		}

		ctx, cancel := context.WithCancel(context.Background())

		go h.handleIncoming(cancel, conn, room)
		go h.handleOutgoing(ctx, conn, room)

		h.l.Info("wss connected")
	}
}

type Host struct {
	l   logs.Logger
	UID string
}

func (h *Host) handleOutgoing(ctx context.Context, conn *websocket.Conn, room *Room) {
	ticker := time.NewTicker(_pingPeriod)
	defer func() {
		ticker.Stop()
		conn.Close()
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-room.HostMsg:
			conn.SetWriteDeadline(time.Now().Add(_writeWait))
			if !ok {
				// The hub closed the channel.
				conn.WriteMessage(websocket.CloseMessage, []byte{})
				h.l.Warn("hostMsg channel closed")
				return
			}

			w, err := conn.NextWriter(websocket.TextMessage)
			if err != nil {
				h.l.Errorf("NextWriter TextMessage, err: %+v", err)
				return
			}

			data, err := json.Marshal(msg)
			if err != nil {
				h.l.Errorf("message.Marshal, err: %+v", err)
				continue
			} else {
				w.Write(data)
			}

			if err := w.Close(); err != nil {
				h.l.Errorf("w.Close, err: %+v", err)
				return
			}

			h.l.Debug("outgoing message sent")
		case <-ticker.C:
			conn.SetWriteDeadline(time.Now().Add(_writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				h.l.Errorf("WriteMessage, err: %+v", err)
				return
			}
		}
	}
}

func (h *Host) handleIncoming(cancel context.CancelFunc, conn *websocket.Conn, room *Room) {
	defer func() {
		conn.Close()
		cancel()
	}()
	conn.SetReadLimit(_maxMessageSize)
	conn.SetReadDeadline(time.Now().Add(_pongWait))
	conn.SetPongHandler(func(string) error { conn.SetReadDeadline(time.Now().Add(_pongWait)); return nil })

	for {
		msgType, message, err := conn.ReadMessage()
		h.l.Debug("message received: ", "type: ", msgType, ", message: ", string(message))

		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				h.l.Errorf("websocket.IsUnexpectedCloseError, err: %+v", err)
			}

			break
		}

		if msgType == -1 {
			break
		}

		var msg HostWsMessageIncoming
		if len(message) != 0 {
			if err := json.Unmarshal(message, &msg); err != nil {
				h.l.Errorf("json.Unmarshal, err: %+v", err)

				continue
			}
		}

		h.handleHostIncomingMessage(room, msg)
	}
}

func (h *Host) handleHostIncomingMessage(room *Room, msg HostWsMessageIncoming) {
	if msg.Connect {
		h.handleConnect(room)
	}

	if msg.SetGame != nil {
		h.handleSetGame(room, msg.SetGame)
	}

	if msg.Round != nil {
		h.handleRound(room, msg.Round)
	}
}

func (h *Host) handleConnect(room *Room) {
	h.l.Debug("handleConnect")
	room.HostMsg <- HostWsMessageOutgoing{
		Connect: &HostWsMessageConnectResponse{
			Dashboard: room.GetDashboard(),
			Round:     room.Round.Load(),
			EndTime:   room.RoundEndTime.Load(),
			GameOver:  room.IsGameOver.Load(),
		},
		Timestamp: time.Now().UnixMilli(),
	}
}

func (h *Host) handleSetGame(room *Room, msg *HostWsMessageSetGameIncoming) {
	h.l.Debug("handleSetGame")
	m := make(map[string]*Candidate, len(msg.Candidates))

	order := 0
	for _, candidate := range msg.Candidates {
		if len(candidate.Name) == 0 {
			continue
		}

		candidate.ID = uuid.NewString()
		candidate.Order = order
		m[candidate.ID] = candidate
		order++
	}

	room.StoreCandidates(m)

	cs := room.GetCandidates()
	ds := room.GetDashboard()

	room.BroadcastPlayers(PlayerWsMessageOutgoing{
		Connect: &PlayerWsMessageConnectResponse{
			Candidates: cs,
			Dashboard:  ds,
			GameOver:   room.IsGameOver.Load(),
		},
		Timestamp: time.Now().UnixMilli(),
	})
}

func (h *Host) handleRound(room *Room, msg *HostWsMessageRoundIncoming) {
	h.l.Debug("handleRound")

	var endTime int64
	if room.Round.Load() > msg.Round {
		h.l.Warnf("skip round, saved: %d, incoming: %d", room.Round.Load(), msg.Round)
	} else {
		switch {
		case msg.GameOver:
			alreadyGameOver := room.IsGameOver.Swap(true)
			if alreadyGameOver {
				h.l.Warn("skip round, already game over")
				return
			}
			room.BroadcastDashboardUpdate(true)
		case room.IsGameOver.Load():
			h.l.Warn("skip round, game over")
			return
		case msg.Start:
			endTime = time.Now().Add(_countdownDuration).UnixMilli()
			room.Round.Store(msg.Round + 1)
			room.RoundEndTime.Store(endTime)
		default:
			h.l.Warn("skip round, unknown")
			return
		}
	}

	dashboard := room.GetDashboard()
	gameOver := room.IsGameOver.Load()
	round := room.Round.Load()

	room.HostMsg <- HostWsMessageOutgoing{
		Round: &HostWsMessageRoundResponse{
			Round:    round,
			GameOver: gameOver,
			EndTime:  endTime,
		},
		Dashboard: &HostWsMessageDashboardResponse{
			Dashboard: dashboard,
			GameOver:  gameOver,
		},
		Timestamp: time.Now().UnixMilli(),
	}

	room.BroadcastPlayers(PlayerWsMessageOutgoing{
		Round: &PlayerWsMessageRoundResponse{
			Dashboard: dashboard,
			Round:     round,
			EndTime:   endTime,
			GameOver:  gameOver,
		},
		Timestamp: time.Now().UnixMilli(),
	})
}
