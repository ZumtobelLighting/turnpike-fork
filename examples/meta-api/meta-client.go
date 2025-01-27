package main

import (
	"log"

	"gopkg.in/beatgammit/turnpike.v2"
)

func main() {
	c, err := turnpike.NewWebsocketClient(turnpike.JSON, "ws://localhost:8000/", nil)
	if err != nil {
		log.Fatal(err)
	}
	log.Info("connected to router")
	_, err = c.JoinRealm("turnpike.examples", nil)
	if err != nil {
		log.Fatal(err)
	}
	log.Info("joined realm")
	c.ReceiveDone = make(chan bool)

	onJoin := func(args []interface{}, kwargs map[string]interface{}) {
		log.Info("session joined:", args[0])
	}
	if err := c.Subscribe("wamp.session.on_join", nil, onJoin); err != nil {
		log.Fatalln("Error subscribing to channel:", err)
	}

	onLeave := func(args []interface{}, kwargs map[string]interface{}) {
		log.Info("session left:", args[0])
	}
	if err := c.Subscribe("wamp.session.on_leave", nil, onLeave); err != nil {
		log.Fatalln("Error subscribing to channel:", err)
	}

	log.Info("listening for meta events")
	<-c.ReceiveDone
	log.Info("disconnected")
}
