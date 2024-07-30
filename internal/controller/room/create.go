package room

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
)

type CreateRoomRequest struct {
	UID       string `json:"uid"`
	RoomTitle string `json:"room_title"`
}

type CreateRoomResponse struct {
	RoomID string `json:"room_id"`
}

func CreateRoom() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		buf, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))

			return
		}

		var request CreateRoomRequest
		if err := json.Unmarshal(buf, &request); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))

			return
		}

		slog.Info("CreateRoom", "request", request)

		room := NewRoom(request.UID, request.RoomTitle)

		_roomPool.Store(room.RoomID, room)

		response, err := json.Marshal(CreateRoomResponse{
			RoomID: room.RoomID,
		})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))

			return
		}

		w.Write(response)
	}
}
