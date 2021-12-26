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

func wsHandler(nc *nats.Conn, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade:", err)
		return
	}
	go wsDriver(nc, conn)
}
func wsDriver(nc *nats.Conn, conn *websocket.Conn) {
	address, err := awaitAddress(conn)
	if err != nil {
		log.Println("await address:", err)
		return
	}
	conn.WriteJSON(protocol.AuthResponse{AuthenticatedUntil: 1})

	go func() {
		defer func() {
			log.Println("killing read pump")
		}()
		for {
			mt, message, err := conn.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			log.Printf("recv: %q %s", mt, message)
			switch mt {
			case websocket.TextMessage:
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
			case websocket.BinaryMessage:
				log.Println("unexpected binary message", message)
				return
			case websocket.CloseMessage:
				log.Println("client requesting close", message)
				return
			case websocket.PingMessage:
				log.Println("client ping", message)
				return
			case websocket.PongMessage:
				log.Println("ignoring client pong", message)
			}
		}
	}()

	go func() {
		defer func() {
			log.Println("killing write pump")
		}()
		js, err := nc.JetStream()
		if err != nil {
			log.Fatalf("could not connect to jetstream: %s", err)
		}

		subj := fmt.Sprintf("MESSAGES.%s", address)
		sub, err := js.SubscribeSync(subj, nats.DeliverAll())
		if err != nil {
			log.Fatalf("could not subscribe to MESSAGES: %s", err)
		}
		log.Println("subscribed to:", subj)
		for {
			message, err := sub.NextMsg(1 * time.Second)
			if err == nats.ErrTimeout {
				continue
			}
			if err != nil {
				log.Fatalf("unexpected error reading from nats: %s", err)
			}
			meta, err := message.Metadata()
			if err != nil {
				log.Fatalf("unexpected metadata error: %s", err)
			}
			log.Println("received nats message:", meta)
			var msg protocol.Message
			if err := json.Unmarshal(message.Data, &msg); err != nil {
				log.Fatalf("could not decode message: %s", err)
			}
			if err := conn.WriteJSON(msg); err != nil {
				log.Println("write to websocket:", err)
				return
			}
			if err := message.AckSync(); err != nil {
				log.Println("ack:", err)
			}
		}
	}()
}

func awaitAddress(conn *websocket.Conn) (string, error) {
	for {
		mt, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			return "", err
		}
		log.Printf("recv: %q %s", mt, message)
		switch mt {
		case websocket.TextMessage:
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
		case websocket.BinaryMessage:
			log.Println("unexpected binary message", message)
			return "", fmt.Errorf("unexpected binary message")
		case websocket.CloseMessage:
			log.Println("client requesting close", message)
			return "", fmt.Errorf("client closed without authenticating")
		case websocket.PingMessage:
			log.Println("client ping", message)
			conn.WriteMessage(websocket.PongMessage, []byte{})
		case websocket.PongMessage:
			log.Println("ignoring client pong", message)
		}
	}
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
		wsHandler(nc, w, r)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	address := fmt.Sprintf("0.0.0.0:%s", port)
	log.Printf("listening on %s...\n", address)
	log.Fatal(http.ListenAndServe(address, nil))
}
