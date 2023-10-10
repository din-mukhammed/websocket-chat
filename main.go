package main

import (
	"net/http"
	"sync/atomic"

	log "github.com/mgutz/logxi/v1"

	"github.com/gorilla/websocket"
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

type Message struct {
	Type   int
	Data   []byte
	Sender int
}

/*
in1  \              / out1
in2  -->  fanIn  -->  out2
in3  /              \ out3
*/

func fanIn(msgs <-chan Message, consumers []chan Message) {
	for msg := range msgs {
		log.Info("fanIn", "msg", msg.Data, "cc", len(consumers))
		for i, out := range consumers {
			if msg.Sender == i {
				continue
			}
			if out == nil {
				continue
			}
			log.Info("fanIn write to consumer")
			out <- msg
		}
	}
}

func reader(id int, conn *websocket.Conn, out chan<- Message) {
	for {
		t, msg, err := conn.ReadMessage()
		if err != nil {
			if t == websocket.CloseMessage {
				return
			}
			log.Error("error on read msg", "err", err)
			return
		}
		log.Info("read", "msg", msg)
		out <- Message{t, msg, id}
	}
}

var (
	defaultRoom = newRoom(10)
)

type room struct {
	msgs   chan Message
	actors []chan Message

	id int64
}

func newRoom(capacity int) *room {
	return &room{
		actors: make([]chan Message, capacity),
		msgs:   make(chan Message),
	}
}

func (r *room) add(conn *websocket.Conn) {
	out := make(chan Message)
	ind := atomic.LoadInt64(&r.id)
	r.actors[ind] = out
	atomic.AddInt64(&r.id, 1)
	go func() {
		defer conn.Close()
		// TODO: remove from consumers

		reader(int(ind), conn, r.msgs)
	}()
	go writer(conn, out)
}

func writer(conn *websocket.Conn, msgs <-chan Message) {
	for m := range msgs {
		if err := conn.WriteMessage(m.Type, m.Data); err != nil {
			log.Error("error on write", "err", err)
		}
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error("error on upgrade connection", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	defaultRoom.add(ws)
}

func main() {
	go fanIn(defaultRoom.msgs, defaultRoom.actors)
	http.HandleFunc("/", handler)

	h := ":8080"
	log.Info("starting web server", "host", h)
	if err := http.ListenAndServe(h, nil); err != nil {
		log.Fatal(err.Error())
	}
}
