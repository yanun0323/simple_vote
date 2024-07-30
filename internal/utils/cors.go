package utils

import (
	"net/http"
)

func CORS(fn func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		SetCORS(w)
		fn(w, r)
	})
}

func SetCORS(w http.ResponseWriter) {
	// w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding,Accept-Language, X-CSRF-Token, Authorization, Connection, Host, Origin, User-Agent, Cookie, Upgrade, Sec_Websocket_Extensions, Sec-WebSocket-Key, Sec-WebSocket-Version, Sec-WebSocket-Protocol, Sec-WebSocket-Version, Pragma, Cache-Control")
}
