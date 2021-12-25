package main

import (
	"log"

	"github.com/nats-io/nats.go"
)

func main() {
	// Connect to NATS
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatalf("could not connect to nats: %s", err)
	}
	defer nc.Close()

	// Create JetStream Context
	js, err := nc.JetStream()
	if err != nil {
		log.Fatalf("could not connect to jetstream: %s", err)
	}

	info, err := js.AddStream(&nats.StreamConfig{
		Name:     "ORDERS",
		Subjects: []string{"ORDERS.*"},
	})
	if err != nil {
		log.Fatalf("could not create stream: %s", err)
	}
	log.Printf("created stream: %+v", info)

	ack, err := js.Publish("ORDERS.yo1", []byte("hello"))
	if err != nil {
		log.Fatalf("could not publish: %s", err)
	}
	log.Println("acked publish:", ack)

	log.Println("ORDERS.yo1")
	pull(js, "ORDERS.yo1")

	log.Println("ORDERS.yo2")
	pull(js, "ORDERS.yo2")
}

func pull(js nats.JetStream, subj string) {
	sub, err := js.SubscribeSync(subj, nats.StartSequence(6))
	if err != nil {
		log.Fatalf("could not subscribe: %s", err)
	}
	for i := 0; i < 10; i++ {
		msg, err := sub.NextMsg(0)
		if err != nil {
			log.Fatalf("could not pull msg: %s", err)
		}
		meta, err := msg.Metadata()
		if err != nil {
			log.Fatalf("could not fetch metadata: %s", err)
		}
		log.Printf("consumer %s @ %+v: %v", subj, meta.Sequence.Stream, msg.Data)
	}
}
