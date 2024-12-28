package corewebsocket

import (
	"encoding/base64"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"
)

var (
	HubMap = make(map[uuid.UUID]*Hub)
)

type Hub struct {
	ID        uuid.UUID
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
		case client := <-h.Unregister:
			if _, ok := h.Clients[client]; ok {
				// broadcast leave notification
				msg := Message{
					MsgType: 1,
					UID:     client.ID.String(),
					Data:    "left",
				}
				// wrap by goroutine to avoid deadlock
				go func() {
					if len(h.Clients) < 1 {
						return
					}
					h.Broadcast <- msg
				}()

				// clean up
				delete(h.Clients, client)
				close(client.Send)
				// check if hub should be closed
				if len(h.Clients) == 0 {
					idStr := base64.RawURLEncoding.EncodeToString(h.ID[:])
					slog.Debug("hub closed: no client in hub", "id", idStr)
					return
				}
				// @TODO check host transfer
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
