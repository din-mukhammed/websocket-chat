package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

func reader(conn *websocket.Conn) {
	defer conn.Close()
	for {
		log.Printf(">>> reading msg")
		t, p, err := conn.ReadMessage()
		if err != nil {
			if t == websocket.CloseMessage {
				log.Printf("closing connection")
				return
			}
			log.Printf("!!! wtf on read: %v", err)
			return
		}

		log.Printf("t: %v, p: %v", t, string(p))

		if err := conn.WriteMessage(t, p); err != nil {
			log.Printf("!!! wtf on write: %v", err)
			return
		}
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}

	reader(ws)
}

func main() {
	http.HandleFunc("/", handler)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
