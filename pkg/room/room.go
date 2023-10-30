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
	Id          uuid.UUID
	Messages    chan Message
	IsConnected bool
}

type Room struct {
	msgs chan Message

	mu      sync.RWMutex
	actors  map[int]*Client
	monoInd int
}

func New(capacity int) *Room {
	return &Room{
		actors: make(map[int]*Client, capacity),
		msgs:   make(chan Message),
	}
}

func (r *Room) Messages() chan Message {
	return r.msgs
}

func (r *Room) Subscribers() []*Client {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var cc []*Client
	for _, c := range r.actors {
		if c.IsConnected {
			cc = append(cc, c)
		}
	}

	return cc
}

func (r *Room) AddClient(conn *websocket.Conn) {
	done := make(chan struct{})
	c := &Client{
		Id:          uuid.NewV4(),
		Messages:    make(chan Message),
		IsConnected: true,
	}

	userInd := 0
	r.mu.Lock()
	r.actors[r.monoInd] = c
	userInd = r.monoInd
	r.monoInd++
	r.mu.Unlock()

	go func() {
		defer conn.Close()
		defer func() {
			done <- struct{}{}
			r.removeSub(userInd)
		}()

		reader(c.Id, conn, r.msgs)
	}()
	go writer(conn, c, done)
}

func (r *Room) removeSub(ind int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.actors[ind].IsConnected = false
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
