package room

import (
	"log/slog"
	"net/http"

	"main/internal/keyword"
	"main/internal/utils"

	"github.com/google/uuid"
	"github.com/spf13/viper"
)

func GetRoom() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		room, ok := _roomPool.Load(r.PathValue("room_id"))
		if !ok {
			slog.Warn("GetRoom, room id not found in pool", "room_id", r.PathValue("room_id"))
			w.Write([]byte("room not found"))

			return
		}

		isRoomStarted := room.IsGameStart.Load()
		if isRoomStarted {
			slog.Warn("GetRoom, room is already started", "room_id", r.PathValue("room_id"))
		}

		w.Write(utils.ReadFile("./internal/resource/room.html", map[string]string{
			keyword.Host:        viper.GetString("host"),
			keyword.UID:         uuid.NewString(),
			keyword.RoomID:      room.RoomID,
			keyword.RoomStarted: utils.BoolToString(isRoomStarted),
			keyword.RoomTitle:   room.Title,
		}))
	}
}
