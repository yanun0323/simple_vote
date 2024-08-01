package room

import (
	"sort"
	"time"

	"main/internal/utils"

	"github.com/yanun0323/pkg/logs"
)

var (
	_roomPool           = utils.NewSyncMap[string, *Room]()
	_defaultChannelSize = 500
)

type Room struct {
	l            logs.Logger
	RoomID       string
	Title        string
	PlayerUpdate chan struct{}
	HostMsg      chan HostWsMessageOutgoing
	Round        *utils.SyncValue[int]
	RoundEndTime *utils.SyncValue[int64]
	IsGameStart  *utils.SyncValue[bool]
	IsGameOver   *utils.SyncValue[bool]
	playerTable  *utils.SyncMap[string, *Player]
	dashboard    *utils.SyncMap[string, *Candidate]
	nickname     *utils.NicknamePool
}

func NewRoom(roomID string, title string) *Room {
	return &Room{
		l:            logs.New(logs.LevelDebug).WithField("room", roomID),
		RoomID:       roomID,
		Title:        title,
		PlayerUpdate: make(chan struct{}, _defaultChannelSize),
		HostMsg:      make(chan HostWsMessageOutgoing, _defaultChannelSize),
		Round:        utils.NewSyncValue(0),
		RoundEndTime: utils.NewSyncValue[int64](0),
		IsGameStart:  utils.NewSyncValue(false),
		IsGameOver:   utils.NewSyncValue(false),
		playerTable:  utils.NewSyncMap[string, *Player](),
		dashboard:    utils.NewSyncMap[string, *Candidate](),
		nickname:     utils.NewNicknamePool(),
	}
}

func (r *Room) BroadcastDashboardUpdate(skipHost ...bool) {
	sli := r.playerTable.ValueSlice()
	for _, player := range sli {
		player.Channel <- PlayerWsMessageOutgoing{
			Dashboard: &PlayerWsMessageDashboardResponse{
				Dashboard: r.GetDashboard(),
				GameOver:  r.IsGameOver.Load(),
			},
			Timestamp: time.Now().UnixMilli(),
		}
	}

	if len(skipHost) != 0 && skipHost[0] {
		return
	}

	r.HostMsg <- HostWsMessageOutgoing{
		Dashboard: &HostWsMessageDashboardResponse{
			Dashboard: r.GetDashboard(),
			GameOver:  r.IsGameOver.Load(),
		},
		Timestamp: time.Now().UnixMilli(),
	}
}

func (r *Room) BroadcastPlayers(msg PlayerWsMessageOutgoing) {
	sli := r.playerTable.ValueSlice()
	for _, player := range sli {
		player.Channel <- msg
	}
}

func (r *Room) AddPlayer(player *Player) {
	if name, ok := r.nickname.Take(); ok {
		player.Name = name
	} else {
		player.Name = player.UID
	}
	r.playerTable.Store(player.UID, player)
}

func (r *Room) GetPlayer(uid string) (*Player, bool) {
	return r.playerTable.Load(uid)
}

func (r *Room) GetPlayerNames() []string {
	sli := r.playerTable.ValueSlice()
	names := make([]string, 0, _defaultChannelSize)
	for _, player := range sli {
		if player.Online.Load() {
			names = append(names, player.Name)
		}
	}

	return names
}

func (r *Room) GetDashboard() []*Candidate {
	d := r.dashboard.ValueSlice()

	sort.Slice(d, func(i, j int) bool {
		if d[i].Score != d[j].Score {
			return d[i].Score > d[j].Score
		}

		return d[i].Order < d[j].Order
	})

	return d
}

func (r *Room) GetCandidates() []*Candidate {
	d := r.dashboard.ValueSlice()

	sort.Slice(d, func(i, j int) bool {
		return d[i].Order < d[j].Order
	})

	return d
}

func (r *Room) StoreCandidates(cds map[string]*Candidate) {
	r.dashboard.Stores(cds)
}

func (r *Room) VoteCandidate(uid string, round int, candidate string) {
	l := r.l.WithField("voter", uid).WithField("round", round).WithField("candidate", candidate)
	if r.IsGameOver.Load() {
		l.Debug("game over, skip voting")
		return
	}

	if r.Round.Load() != round {
		l.Debug("round not match, skip voting")
		return
	}

	player, ok := r.GetPlayer(uid)
	if !ok {
		l.Debug("player not found, skip voting")
		return
	}

	voted, ok := player.VoteTable.Load(round)
	if ok && len(voted) != 0 {
		l.Debug("round already voted, skip voting")
		return
	}

	r.dashboard.Do(candidate, func(d *Candidate) {
		r.l.Debug("candidate: ", candidate, ", uid: ", uid, ", round: ", round)
		if d.ID != candidate {
			l.Debug("candidate not found, skip voting")
			return
		}

		d.Score++
	})

	player.VoteTable.Store(round, candidate)

	r.BroadcastDashboardUpdate()
}
