package actor

import (
	"log"
	"sync"

	"address.chat/api/protocol"
)

type Router struct {
	lock   sync.Mutex
	actors map[string]*Actor
}

func NewRouter() *Router {
	return &Router{
		lock:   sync.Mutex{},
		actors: map[string]*Actor{},
	}
}

func (router *Router) Get(address string) *Actor {
	router.lock.Lock()
	defer router.lock.Unlock()
	a, ok := router.actors[address]
	if !ok {
		a = newActor()
		go a.loop()
		router.actors[address] = a
	}
	return a
}

type Actor struct {
	Incoming    chan protocol.SendRequest
	lock        *sync.Mutex
	nextId      int
	subscribers map[int]chan protocol.SendRequest
}

func newActor() *Actor {
	return &Actor{
		Incoming:    make(chan protocol.SendRequest),
		lock:        &sync.Mutex{},
		nextId:      0,
		subscribers: map[int]chan protocol.SendRequest{},
	}
}

func (a *Actor) loop() {
	for m := range a.Incoming {
		a.fanout(m)
	}
}

func (a *Actor) Subscribe() chan protocol.SendRequest {
	a.lock.Lock()
	defer a.lock.Unlock()
	ch := make(chan protocol.SendRequest)
	a.subscribers[a.nextId] = ch
	a.nextId++
	return ch
}

func (a *Actor) fanout(m protocol.SendRequest) {
	a.lock.Lock()
	defer a.lock.Unlock()
	for id, ch := range a.subscribers {
		select {
		case ch <- m:
			log.Println("fanned out to subscriber:", id)
		default:
			log.Println("dead subscriber:", id)
			delete(a.subscribers, id)
			close(ch)
		}
	}
}
