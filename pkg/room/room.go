package room

import (
	"sync"

	log "github.com/mgutz/logxi/v1"
	uuid "github.com/satori/go.uuid"

	"github.com/gorilla/websocket"
)

type Message struct {
	Type   int
	Data   []byte
	Sender uuid.UUID
}

type Client struct {
	Id       uuid.UUID
	Messages chan Message
}

type Room struct {
	msgs chan Message

	mu     sync.RWMutex
	actors []*Client
}

func New(capacity int) *Room {
	return &Room{
		actors: make([]*Client, 0, capacity),
		msgs:   make(chan Message),
	}
}

func (r *Room) Messages() chan Message {
	return r.msgs
}

func (r *Room) Subscribers() []*Client {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.actors
}

func (r *Room) AddClient(conn *websocket.Conn) {
	done := make(chan struct{})
	c := &Client{
		Id:       uuid.NewV4(),
		Messages: make(chan Message),
	}

	r.mu.Lock()
	r.actors = append(r.actors, c)
	ind := len(r.actors) - 1
	r.mu.Unlock()

	go func() {
		i := ind
		defer conn.Close()
		defer func() {
			done <- struct{}{}
			r.removeSub(i)
		}()

		reader(c.Id, conn, r.msgs)
	}()
	go writer(conn, c, done)
}

func (r *Room) removeSub(ind int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := len(r.actors)
	if n == 0 {
		return
	}
	r.actors[ind] = r.actors[n-1]
	r.actors = r.actors[:n-1]
}

func writer(conn *websocket.Conn, client *Client, done <-chan struct{}) {
	for {
		select {
		case m := <-client.Messages:
			if err := conn.WriteMessage(m.Type, m.Data); err != nil {
				log.Error("error on write", "err", err)
			}
		case <-done:
			log.Info("done writer()")
			return
		}
	}
}

func reader(id uuid.UUID, conn *websocket.Conn, out chan<- Message) {
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
