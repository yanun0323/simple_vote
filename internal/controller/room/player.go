package room

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"main/internal/utils"

	"github.com/gorilla/websocket"
	"github.com/yanun0323/pkg/logs"
)

var _upgrade = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type Candidate struct {
	ID    string `json:"id"`
	Order int    `json:"order"`
	Name  string `json:"name"`
	Score int    `json:"score"`
}

type Player struct {
	l         logs.Logger
	UID       string
	Name      string
	Online    *utils.SyncValue[bool]
	Channel   chan PlayerWsMessageOutgoing
	VoteTable *utils.SyncMap[int, string]
}

func NewPlayer(uid string, name string) *Player {
	return &Player{
		l:         logs.New(logs.LevelDebug).WithField("player", uid),
		UID:       uid,
		Name:      name,
		Online:    utils.NewSyncValue(false),
		Channel:   make(chan PlayerWsMessageOutgoing, _defaultChannelSize),
		VoteTable: utils.NewSyncMap[int, string](),
	}
}

type PlayerWsMessageIncoming struct {
	Connect bool                         `json:"connect"`
	Vote    *PlayerWsMessageVoteIncoming `json:"vote"`
}

type PlayerWsMessageVoteIncoming struct {
	Round     int    `json:"round"`
	Candidate string `json:"candidate"`
}

type PlayerWsMessageOutgoing struct {
	Connect   *PlayerWsMessageConnectResponse   `json:"connect,omitempty"`
	Round     *PlayerWsMessageRoundResponse     `json:"round,omitempty"`
	Dashboard *PlayerWsMessageDashboardResponse `json:"dashboard,omitempty"`
	Timestamp int64                             `json:"timestamp"`
}

type (
	PlayerWsMessageConnectResponse struct {
		Candidates []*Candidate `json:"candidates"`
		Dashboard  []*Candidate `json:"dashboard"`
		Round      int          `json:"round"`
		RoundVoted string       `json:"round_voted"`
		EndTime    int64        `json:"end_time"`
		GameOver   bool         `json:"game_over"`
		PlayerName string       `json:"player_name"`
	}

	PlayerWsMessageRoundResponse struct {
		Dashboard []*Candidate `json:"dashboard"`
		Round     int          `json:"round"`
		EndTime   int64        `json:"end_time"`
		GameOver  bool         `json:"game_over"`
	}

	PlayerWsMessageDashboardResponse struct {
		Dashboard []*Candidate `json:"dashboard"`
		GameOver  bool         `json:"game_over"`
	}
)

func ConnectPlayer() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logs.New(logs.LevelDebug).WithField("player", r.RequestURI)
		l.Info("wss request received")
		utils.SetWss(r)

		roomID, uid := r.PathValue("room_id"), r.PathValue("uid")
		if roomID == "" || uid == "" {
			l.Warn("roomID or uid is empty")
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		l = logs.New(logs.LevelDebug).WithField("player", uid)

		room, ok := _roomPool.Load(roomID)
		if !ok {
			l.Warn("room not found")
			w.WriteHeader(http.StatusNotFound)

			return
		}

		conn, err := _upgrade.Upgrade(w, r, nil)
		if err != nil {
			l.Errorf("upgrade, err: %+v", err)
		}

		player, ok := room.GetPlayer(uid)
		if !ok {
			l.Warn("player not found")
			w.WriteHeader(http.StatusNotFound)

			return
		}

		player.Online.Store(true)
		room.PlayerUpdate <- struct{}{}
		ctx, cancel := context.WithCancel(context.Background())

		go player.handlePlayerIncoming(cancel, conn, room)
		go player.handlePlayerOutgoing(ctx, conn)
		player.l.Info("wss connected")
	}
}

func (p *Player) handlePlayerOutgoing(ctx context.Context, conn *websocket.Conn) {
	ticker := time.NewTicker(_pingPeriod)
	defer func() {
		ticker.Stop()
		conn.Close()
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-p.Channel:
			if msg.Connect != nil && len(msg.Connect.Dashboard) > 3 {
				msg.Connect.Dashboard = msg.Connect.Dashboard[:3]
			}

			if msg.Round != nil && len(msg.Round.Dashboard) > 3 {
				msg.Round.Dashboard = msg.Round.Dashboard[:3]
			}

			if msg.Dashboard != nil && len(msg.Dashboard.Dashboard) > 3 {
				msg.Dashboard.Dashboard = msg.Dashboard.Dashboard[:3]
			}

			conn.SetWriteDeadline(time.Now().Add(_writeWait))
			if !ok {
				// The hub closed the channel.
				conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := conn.NextWriter(websocket.TextMessage)
			if err != nil {
				p.l.Errorf("NextWriter, err: %+v", err)
				return
			}

			data, err := json.Marshal(msg)
			if err != nil {
				p.l.Errorf("message.Marshal, err: %+v", err)
				continue
			} else {
				w.Write(data)
			}

			if err := w.Close(); err != nil {
				p.l.Errorf("w.Close, err: %+v", err)
				return
			}
		case <-ticker.C:
			conn.SetWriteDeadline(time.Now().Add(_writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				p.l.Errorf("WriteMessage, err: %+v", err)
				return
			}
		}
	}
}

func (p *Player) handlePlayerIncoming(cancel context.CancelFunc, conn *websocket.Conn, room *Room) {
	defer func() {
		conn.Close()
		cancel()
		p.Online.Store(false)
		room.PlayerUpdate <- struct{}{}
	}()
	conn.SetReadLimit(_maxMessageSize)
	conn.SetReadDeadline(time.Now().Add(_pongWait))
	conn.SetPongHandler(func(string) error { conn.SetReadDeadline(time.Now().Add(_pongWait)); return nil })

	for {
		msgType, message, err := conn.ReadMessage()
		p.l.Debug("message received: ", string(message))

		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				p.l.Errorf("websocket.IsUnexpectedCloseError, err: %+v", err)
			}

			break
		}

		if msgType == -1 {
			p.l.Warn("disconnected")
			break
		}

		var msg PlayerWsMessageIncoming
		if len(message) != 0 {
			if err := json.Unmarshal(message, &msg); err != nil {
				p.l.Errorf("json.Unmarshal, err: %+v", err)

				continue
			}
		}

		p.handlePlayerIncomingMessage(room, msg)

	}
}

func (p *Player) handlePlayerIncomingMessage(room *Room, msg PlayerWsMessageIncoming) {
	if msg.Connect {
		p.handlePlayerConnect(room)
	}

	if msg.Vote != nil {
		p.handlePlayerVote(room, msg.Vote)
	}
}

func (p *Player) handlePlayerConnect(room *Room) {
	round := room.Round.Load()
	voted, _ := p.VoteTable.Load(round)
	p.Channel <- PlayerWsMessageOutgoing{
		Connect: &PlayerWsMessageConnectResponse{
			Candidates: room.GetCandidates(),
			Dashboard:  room.GetDashboard(),
			Round:      round,
			RoundVoted: voted,
			EndTime:    room.RoundEndTime.Load(),
			GameOver:   room.IsGameOver.Load(),
			PlayerName: p.Name,
		},
		Timestamp: time.Now().UnixMilli(),
	}
}

func (p *Player) handlePlayerVote(room *Room, msg *PlayerWsMessageVoteIncoming) {
	room.VoteCandidate(p.UID, msg.Round, msg.Candidate)
}
