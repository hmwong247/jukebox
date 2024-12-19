package corewebsocket

import (
	"encoding/base64"
	"log"

	"github.com/google/uuid"
)

var (
	HubMap = make(map[uuid.UUID]*Hub)
)

type Hub struct {
	ID        uuid.UUID
	Clients   map[*Client]int // multiple host is allowed
	Broadcast chan []byte

	// control channel
	Register   chan *Client
	Unregister chan *Client
}

func NewHub(id uuid.UUID) *Hub {
	clients := make(map[*Client]int)
	// // the first client is the host by default
	// clients[client] = 7

	return &Hub{
		ID:         id,
		Clients:    clients,
		Broadcast:  make(chan []byte),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	defer func() {
		// close all channels
		close(h.Register)
		close(h.Unregister)
		close(h.Broadcast)
		delete(HubMap, h.ID)
	}()
	for {
		select {
		case client := <-h.Register:
			h.Clients[client] = client.Permission
		case client := <-h.Unregister:
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				close(client.Send)
				// check if hub should be closed
				if len(h.Clients) == 0 {
					// h.CloseHub()
					idStr := base64.RawURLEncoding.EncodeToString(h.ID[:])
					log.Printf("hub closed: no client in hub [%v]\n", idStr)
					return
				}
				// @TODO check host transfer
			}
		case msg := <-h.Broadcast:
			// log.Printf("ws msg - client: %v, msg: %v\n", )
			for client := range h.Clients {
				select {
				case client.Send <- msg:
				default:
					close(client.Send)
					delete(h.Clients, client)
				}
			}
		}
	}
}
