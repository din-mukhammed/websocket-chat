package main

import (
	"websocket-chat/pkg/room"

	log "github.com/mgutz/logxi/v1"
)

var (
	// TODO: don't use global vars
	DefaultRoom = room.New(0)
)

/*
in1  \              / out1
in2  -->  fanIn  -->  out2
in3  /              \ out3
*/
func fanIn(r *room.Room) {
	for msg := range r.Messages() {
		ss := r.Subscribers()
		log.Info("fanIn", "msg", msg.Data, "ss", len(ss))
		for _, s := range ss {
			if msg.Sender == s.Id {
				continue
			}
			log.Info("fanIn write to consumer")
			s.Messages <- msg
		}
	}
}

func main() {
	go fanIn(DefaultRoom)
	h := ":8080"
	web := NewServer(h)

	log.Info("starting web server", "host", h)
	if err := web.ListenAndServe(); err != nil {
		log.Fatal(err.Error())
	}
}
