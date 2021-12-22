package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"address.chat/api/auth"
	"address.chat/api/protocol"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade:", err)
		return
	}
	defer c.Close()
	if err := wsDriver(c); err != nil {
		c.WriteMessage(websocket.TextMessage, []byte(err.Error()))
	}
}
func wsDriver(c *websocket.Conn) error {
	var address string
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			return err
		}
		log.Printf("recv: %q %s", mt, message)
		switch mt {
		case websocket.TextMessage:
			if address == "" {
				var payload protocol.AuthRequest
				if err := json.Unmarshal(message, &payload); err != nil {
					return fmt.Errorf("invalid AuthRequest: %s", err)
				}
				if err := auth.VerifySignature(payload.Address, payload.Challenge, payload.Signature); err != nil {
					return fmt.Errorf("could not verify signature: %s", err)
				}
				address = payload.Address
			} else {
				var payload protocol.SendRequest
				if err := json.Unmarshal(message, &payload); err != nil {
					return fmt.Errorf("invalid SendRequest: %s", err)
				}
				log.Println("user asked us to send a message", payload)
			}
		case websocket.BinaryMessage:
			log.Println("unexpected binary message", message)
			return fmt.Errorf("unexpected binary message")
		case websocket.CloseMessage:
			log.Println("client requesting close", message)
			return nil
		case websocket.PingMessage:
			log.Println("client ping", message)
			c.WriteMessage(websocket.PongMessage, []byte{})
		case websocket.PongMessage:
			log.Println("ignoring client pong", message)
		}
	}

}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok"))
}

func main() {
	const address = "localhost:8080"
	http.HandleFunc("/readyz", healthCheckHandler)
	http.HandleFunc("/alivez", healthCheckHandler)
	http.HandleFunc("/ws", wsHandler)
	log.Printf("listening on %s...\n", address)
	log.Fatal(http.ListenAndServe(address, nil))
}
