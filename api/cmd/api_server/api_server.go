package main

import (
	"encoding/json"
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
	defer conn.Close()
	if err := wsDriver(nc, conn); err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte(err.Error()))
	}
}
func wsDriver(nc *nats.Conn, conn *websocket.Conn) error {
	address, err := awaitAddress(conn)
	if err != nil {
		return err
	}
	conn.WriteJSON(protocol.AuthResponse{AuthenticatedUntil: 1})

	done := false
	send := make(chan protocol.Message)
	recv := make(chan protocol.Message)
	defer close(send)
	defer close(recv)
	defer func() {
		done = true
	}()
	go func() {
		for msg := range send {
			data, err := json.Marshal(msg)
			if err != nil {
				log.Fatalf("could not marshall message: %s", err)
			}
			if err := nc.Publish("MESSAGES", data); err != nil {
				log.Fatalf("could not publish to nats: %s", err)
			}
		}
	}()
	js, err := nc.JetStream()
	if err != nil {
		log.Fatalf("could not connect to jetstream: %s", err)
	}
	sub, err := js.SubscribeSync("MESSAGES", nats.DeliverAll())
	if err != nil {
		log.Fatalf("could not subscribe to MESSAGES: %s", err)
	}
	go func() {
		for {
			message, err := sub.NextMsg(1 * time.Second)
			if done {
				return
			}
			if err == nats.ErrTimeout {
				continue
			}
			if err != nil {
				log.Fatalf("unexpected error reading from nats: %s", err)
			}
			var msg protocol.Message
			if err := json.Unmarshal(message.Data, &msg); err != nil {
				log.Fatalf("could not decode message: %s", err)
			}
			recv <- msg
		}
	}()

	errCh := make(chan error)
	go func() {
		err := readPump(conn, address, send)
		log.Println("read pump:", err)
		select {
		case errCh <- err:
		default:
		}
	}()
	go func() {
		err := writePump(conn, recv)
		log.Println("write pump:", err)
		select {
		case errCh <- err:
		default:
		}
	}()
	return <-errCh
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

func readPump(conn *websocket.Conn, address string, ch chan protocol.Message) error {
	for {
		mt, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			return err
		}
		log.Printf("recv: %q %s", mt, message)
		switch mt {
		case websocket.TextMessage:
			var payload protocol.SendRequest
			if err := json.Unmarshal(message, &payload); err != nil {
				return fmt.Errorf("invalid SendRequest: %s", err)
			}
			log.Println("user asked us to send a message", payload)
			ch <- protocol.Message{
				From:    address,
				To:      payload.To,
				Content: payload.Content,
			}
		case websocket.BinaryMessage:
			log.Println("unexpected binary message", message)
			return fmt.Errorf("unexpected binary message")
		case websocket.CloseMessage:
			log.Println("client requesting close", message)
			return nil
		case websocket.PingMessage:
			log.Println("client ping", message)
			conn.WriteMessage(websocket.PongMessage, []byte{})
		case websocket.PongMessage:
			log.Println("ignoring client pong", message)
		}
	}
}

func writePump(conn *websocket.Conn, ch chan protocol.Message) error {
	for {
		m, ok := <-ch
		// TODO: batch outgoing messages
		if !ok {
			return nil
		}
		if err := conn.WriteJSON(m); err != nil {
			return err
		}
	}
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok"))
}

func main() {
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatalf("could not connect to nats: %s", err)
	}
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
