package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"address.chat/api/actor"
	"address.chat/api/auth"
	"address.chat/api/protocol"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func wsHandler(router *actor.Router, w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade:", err)
		return
	}
	defer c.Close()
	if err := wsDriver(router, c); err != nil {
		c.WriteMessage(websocket.TextMessage, []byte(err.Error()))
	}
}
func wsDriver(router *actor.Router, conn *websocket.Conn) error {
	address, err := awaitAddress(conn)
	if err != nil {
		return err
	}
	conn.WriteJSON(protocol.AuthResponse{AuthenticatedUntil: 1})

	a := router.Get(address)
	errCh := make(chan error)
	go func() {
		err := readPump(conn, a.Incoming)
		log.Println("read pump:", err)
		select {
		case errCh <- err:
		default:
		}
	}()
	go func() {
		ch := a.Subscribe()
		err := writePump(conn, ch)
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

func readPump(conn *websocket.Conn, ch chan protocol.SendRequest) error {
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
			ch <- payload
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

func writePump(conn *websocket.Conn, ch chan protocol.SendRequest) error {
	for {
		m, ok := <-ch
		// TODO: batch outgoing messages
		if !ok {
			return conn.WriteMessage(websocket.CloseMessage, []byte{})
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
	router := actor.NewRouter()
	http.HandleFunc("/readyz", healthCheckHandler)
	http.HandleFunc("/alivez", healthCheckHandler)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		wsHandler(router, w, r)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	address := fmt.Sprintf("0.0.0.0:%s", port)
	log.Printf("listening on %s...\n", address)
	log.Fatal(http.ListenAndServe(address, nil))
}
