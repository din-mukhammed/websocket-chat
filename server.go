package main

import (
	"net/http"

	log "github.com/mgutz/logxi/v1"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

func NewServer(addr string) *http.Server {
	router := mux.NewRouter()
	web := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	router.HandleFunc("/", handler)

	return web
}

func handler(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error("error on upgrade connection", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	DefaultRoom.AddClient(ws)
}
