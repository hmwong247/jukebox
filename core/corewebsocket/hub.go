package corewebsocket

import (
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"math"

	"github.com/google/uuid"
)

var (
	HubMap = make(map[uuid.UUID]*Hub)
)

type Hub struct {
	ID        uuid.UUID
	Host      *Client
	Clients   map[*Client]int // multiple host is allowed
	Broadcast chan Message

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
		Host:       nil,
		Clients:    clients,
		Broadcast:  make(chan Message),
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
			// set host
			if h.Host == nil {
				h.Host = client
			}
		case client := <-h.Unregister:
			if _, ok := h.Clients[client]; ok {
				// broadcast leave notification
				msg := Message{
					MsgType:  1,
					UID:      client.ID.String(),
					Username: client.Name,
					Data:     "left",
				}
				go h.BroadcastMsg(msg)

				// clean up
				delete(h.Clients, client)
				close(client.Send)
				// check if hub should be closed
				if len(h.Clients) == 0 {
					idStr := base64.RawURLEncoding.EncodeToString(h.ID[:])
					slog.Debug("hub closed: no client in hub", "id", idStr)
					return
				} else {
					// check host transfer
					if h.Host.ID == client.ID {
						h.Host = h.NextHost()
						msg := Message{
							MsgType:  1,
							UID:      h.Host.ID.String(),
							Username: h.Host.Name,
							Data:     "host",
						}
						go h.BroadcastMsg(msg)
					}
				}
			}
		case msg := <-h.Broadcast:
			msgJson, err := json.Marshal(msg)
			if err != nil {
				slog.Error("json err", "err", err)
			}
			slog.Debug("ws msg", "cid", msg.UID, "msg", msg.Data)
			for client := range h.Clients {
				select {
				case client.Send <- []byte(msgJson):
				default:
					close(client.Send)
					delete(h.Clients, client)
				}
			}
		}
	}
}

func (h *Hub) NextHost() *Client {
	// find the next client joined after the current host
	var min int64 = math.MaxInt64
	var match *Client
	for client := range h.Clients {
		if client.ID == h.Host.ID {
			continue
		}
		if client.JoinUnixMilli < min {
			min = client.JoinUnixMilli
			match = client
		}
	}
	return match
}

func (h *Hub) BroadcastMsg(msg Message) {
	// wrap by goroutine to avoid deadlock
	if len(h.Clients) < 1 {
		return
	}
	h.Broadcast <- msg

}
