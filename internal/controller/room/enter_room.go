package room

import (
	"bytes"
	"encoding/base64"
	"log/slog"
	"net/http"

	"main/internal/keyword"
	"main/internal/utils"

	"github.com/spf13/viper"
	"github.com/yeqown/go-qrcode"
)

func EnterRoom() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		room, ok := _roomPool.Load(r.PathValue("room_id"))
		if !ok {
			slog.Warn("GetRoom, room id not found in pool", "room_id", r.PathValue("room_id"))
			w.Write([]byte("room not found"))

			return
		}

		if r.PathValue("uid") != room.RoomID {
			slog.Info("EnterRoom, player", "uid", r.PathValue("uid"))
			// HINT: Player's Page
			w.Write(utils.ReadFile("./internal/resource/player.html", map[string]string{
				keyword.Host:      viper.GetString("host"),
				keyword.RoomID:    room.RoomID,
				keyword.RoomTitle: room.Title,
			}))
			return
		}

		url := viper.GetString("host") + "/vote/" + room.RoomID
		qrc, err := qrcode.New(url, qrcode.WithQRWidth(10))
		if err != nil {
			slog.Error("qrcode.New", "error", err)
			return
		}

		qr := bytes.NewBuffer(nil)
		if err := qrc.SaveTo(qr); err != nil {
			slog.Error("qrcode.SaveTo", "error", err)
			return
		}

		// HINT: Host's Page
		qrcImg := base64.StdEncoding.EncodeToString(qr.Bytes())
		w.Write(utils.ReadFile("./internal/resource/host.html", map[string]string{
			keyword.QRCode:    qrcImg,
			keyword.Host:      viper.GetString("host"),
			keyword.RoomLink:  url,
			keyword.RoomID:    room.RoomID,
			keyword.RoomTitle: room.Title,
		}))
	}
}
