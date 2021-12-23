package actor

import (
	"address.chat/api/protocol"
)

type Hub struct {
	Send        chan protocol.Message
	Subscribe   chan SubscribeRequest
	Unsubscribe chan UnsubscribeRequest
	subs        map[string]map[chan protocol.Message]bool
}
type SubscribeRequest struct {
	Address string
	Ch      chan protocol.Message
}
type UnsubscribeRequest struct {
	Address string
	Ch      chan protocol.Message
}

func NewHub() *Hub {
	return &Hub{
		Send:        make(chan protocol.Message),
		Subscribe:   make(chan SubscribeRequest),
		Unsubscribe: make(chan UnsubscribeRequest),
		subs:        make(map[string]map[chan protocol.Message]bool),
	}
}

func (hub *Hub) Loop() {
	for {
		select {
		case msg := <-hub.Send:
			for _, address := range msg.To {
				if subs, ok := hub.subs[address]; ok {
					for ch := range subs {
						select {
						case ch <- msg:
						default:
							delete(subs, ch)
							close(ch)
						}
					}
				}
			}
		case req := <-hub.Subscribe:
			subs, ok := hub.subs[req.Address]
			if !ok {
				subs = make(map[chan protocol.Message]bool)
				hub.subs[req.Address] = subs
			}
			subs[req.Ch] = true
		case req := <-hub.Unsubscribe:
			if subs, ok := hub.subs[req.Address]; ok {
				if _, ok := subs[req.Ch]; ok {
					delete(subs, req.Ch)
					close(req.Ch)
				}
			}
		}
	}
}
