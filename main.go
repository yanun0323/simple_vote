package main

import (
	"log/slog"
	"net/http"

	"main/internal/controller/homepage"
	"main/internal/controller/room"
	"main/internal/utils"

	"github.com/yanun0323/pkg/config"
)

func main() {
	if err := config.Init("config", true); err != nil {
		slog.Error("config.Init", "err", err.Error())
	}

	http.HandleFunc("GET /vote", utils.CORS(homepage.HomePage()))
	http.HandleFunc("GET /vote/{room_id}", utils.CORS(room.GetRoom()))
	http.HandleFunc("GET /vote/{room_id}/{uid}", utils.CORS(room.EnterRoom()))

	http.HandleFunc("POST /api/vote/{room_id}", utils.CORS(room.CreateRoom()))
	http.HandleFunc("POST /api/vote/{room_id}/{uid}", utils.CORS(room.CreatePlayer()))

	// wss
	http.HandleFunc("/api/vote/{room_id}/{uid}/player", utils.CORS(room.ConnectPlayer()))
	http.HandleFunc("/api/vote/{room_id}/{uid}/host", utils.CORS(room.ConnectHost()))

	// listen on port 8080
	http.ListenAndServe(":8080", nil)
}
