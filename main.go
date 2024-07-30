package main

import (
	"log/slog"
	"net/http"

	"main/internal/controller/homepage"
	"main/internal/controller/room"

	"github.com/yanun0323/pkg/config"
)

func main() {
	if err := config.Init("config", true, "./config", "../config", "../../config"); err != nil {
		slog.Error("config.Init", "err", err.Error())
	}

	http.HandleFunc("GET /vote", homepage.HomePage())
	http.HandleFunc("GET /vote/{room_id}", room.GetRoom())
	http.HandleFunc("GET /vote/{room_id}/{uid}", room.EnterRoom())

	http.HandleFunc("POST /api/vote/{room_id}", room.CreateRoom())
	http.HandleFunc("POST /api/vote/{room_id}/{uid}", room.CreatePlayer())

	// wss
	http.HandleFunc("/api/vote/{room_id}/{uid}/player", room.ConnectPlayer())
	http.HandleFunc("/api/vote/{room_id}/{uid}/host", room.ConnectHost())

	// listen on port 8080
	http.ListenAndServe(":8080", nil)
}
