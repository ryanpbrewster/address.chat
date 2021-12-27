package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"address.chat/api/auth"
	"address.chat/api/protocol"
	"github.com/gorilla/websocket"
	"github.com/nats-io/nats.go"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func readPump(conn *websocket.Conn, ch chan<- []byte) {
	defer close(ch)
	defer log.Println("killing read pump")
	for {
		mt, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("failed to read from websocket:", err)
			return
		}
		log.Printf("recv: %q %s", mt, message)
		switch mt {
		case websocket.TextMessage:
			ch <- message
		case websocket.BinaryMessage:
			log.Println("unexpected binary message", message)
			return
		case websocket.CloseMessage:
			log.Println("client requesting close", message)
			return
		case websocket.PingMessage:
			log.Println("ignoring client ping", message)
		case websocket.PongMessage:
			log.Println("ignoring client pong", message)
		}
	}
}

func writePump(conn *websocket.Conn, ch <-chan []byte) {
	defer log.Println("killing write pump")
	for message := range ch {
		if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
			log.Println("failed to write to websocket:", err)
			return
		}
	}
}

func publishPump(nc *nats.Conn, address string, read <-chan []byte) {
	for message := range read {
		var payload protocol.SendRequest
		if err := json.Unmarshal(message, &payload); err != nil {
			log.Printf("invalid SendRequest: %s", err)
			return
		}
		log.Println("user asked us to send a message", payload)
		msg := protocol.Message{
			SentAt:  time.Now().UTC().UnixMilli(),
			From:    address,
			To:      payload.To,
			Content: payload.Content,
		}
		data, err := json.Marshal(msg)
		if err != nil {
			log.Fatalf("could not marshall msg: %s", err)
		}
		for addr := range msg.Participants() {
			if err := nc.Publish(fmt.Sprintf("MESSAGES.%s", addr), data); err != nil {
				log.Fatalf("could not publish to nats: %s", err)
			}
		}
	}
}

func subscribePump(nc *nats.Conn, address string, write chan []byte, done chan struct{}) {
	defer log.Println("killing subscribe pump")
	defer close(write)

	js, err := nc.JetStream()
	if err != nil {
		log.Fatalf("could not connect to jetstream: %s", err)
	}

	subj := fmt.Sprintf("MESSAGES.%s", address)
	sub, err := js.Subscribe(subj, func(message *nats.Msg) {
		meta, err := message.Metadata()
		if err != nil {
			log.Fatalf("unexpected metadata error: %s", err)
		}
		log.Println("received nats message:", meta)
		var msg protocol.Message
		if err := json.Unmarshal(message.Data, &msg); err != nil {
			log.Fatalf("could not parse message: %s", err)
		}
		sync := protocol.SyncMessage{
			Messages: []protocol.Message{msg},
			Seqno:    meta.Sequence.Stream,
		}
		payload, err := json.Marshal(sync)
		if err != nil {
			log.Fatalf("could not marshall sync message: %s", err)
		}
		write <- payload
		if err := message.AckSync(); err != nil {
			log.Println("ack:", err)
		}
	}, nats.DeliverAll())
	if err != nil {
		log.Fatalf("could not subscribe to MESSAGES: %s", err)
	}
	defer sub.Drain()

	<-done
}

func wsDriver(nc *nats.Conn, conn *websocket.Conn) {
	defer conn.Close()
	signal := make(chan error, 1)
	done := make(chan struct{})
	defer close(done)

	read := make(chan []byte)
	go readPump(conn, read)

	address, err := extractAddress(<-read)
	if err != nil {
		log.Println("verify address:", err)
		signal <- fmt.Errorf("verify address: %s", err)
		return
	}
	if err := conn.WriteJSON(protocol.AuthResponse{AuthenticatedUntil: 1}); err != nil {
		return
	}

	go publishPump(nc, address, read)

	write := make(chan []byte)
	go writePump(conn, write)

	go subscribePump(nc, address, write, done)

	if err := <-signal; err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte(err.Error()))
	}
}

func extractAddress(message []byte) (string, error) {
	var request protocol.AuthRequest
	if err := json.Unmarshal(message, &request); err != nil {
		return "", fmt.Errorf("invalid AuthRequest: %s", err)
	}
	log.Println("AuthRequest:", request)
	var payload protocol.AuthPayload
	if err := json.Unmarshal([]byte(request.Payload), &payload); err != nil {
		return "", fmt.Errorf("invalid AuthRequest.Payload: %s", err)
	}
	if err := auth.VerifySignature(payload.Address, request.Payload, request.Signature); err != nil {
		return "", fmt.Errorf("could not verify signature: %s", err)
	}
	// TODO: check payload.ExpiresAt
	return payload.Address, nil
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok"))
}

func main() {
	natsUrl := flag.String("nats", nats.DefaultURL, "the url for the NATS cluster")
	nc, err := nats.Connect(*natsUrl)
	if err != nil {
		log.Fatalf("could not connect to nats: %s", err)
	}
	js, err := nc.JetStream()
	if err != nil {
		log.Fatalf("could not connect to jetstream: %s", err)
	}
	info, err := js.AddStream(&nats.StreamConfig{
		Name:     "MESSAGES",
		Subjects: []string{"MESSAGES.*"},
	})
	if err != nil {
		log.Fatalf("could not create stream: %s", err)
	}
	log.Println("created stream:", info)

	http.HandleFunc("/readyz", healthCheckHandler)
	http.HandleFunc("/alivez", healthCheckHandler)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("failed to upgrade websocket:", err)
			return
		}
		go wsDriver(nc, conn)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	address := fmt.Sprintf("0.0.0.0:%s", port)
	log.Printf("listening on %s...\n", address)
	log.Fatal(http.ListenAndServe(address, nil))
}
