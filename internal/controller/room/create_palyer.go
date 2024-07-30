package room

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
)

type CreatePlayerRequest struct {
	Name string `json:"name"`
}

func CreatePlayer() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		slog.Info("CreatePlayer", "room_id", r.PathValue("room_id"), "uid", r.PathValue("uid"))
		room, ok := _roomPool.Load(r.PathValue("room_id"))
		if !ok {
			slog.Warn("GetRoom, room id not found in pool", "room_id", r.PathValue("room_id"))
			w.Write([]byte("room not found"))

			return
		}

		uid := r.PathValue("uid")
		if len(uid) == 0 {
			slog.Warn("GetRoom, uid not found in path", "uid", uid)
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		buf, err := io.ReadAll(r.Body)
		if err != nil {
			slog.Warn("GetRoom, read body err", "err", err)
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		var req CreatePlayerRequest
		if err := json.Unmarshal(buf, &req); err != nil {
			slog.Warn("GetRoom, unmarshal err", "err", err)
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		if len(req.Name) == 0 {
			slog.Warn("GetRoom, name not found in request", "name", req.Name)
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		room.AddPlayer(NewPlayer(uid, req.Name))
	}
}
