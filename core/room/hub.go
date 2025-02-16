package room

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"main/core/mq"
	"main/core/ytdlp"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	TIMEOUT_HUB = 10 * time.Second
)

var (
	// rid -> *Hub
	HubMap = make(map[uuid.UUID]*Hub)

	// uid -> *Client
	ClientMap      = make(map[uuid.UUID]*Client)
	ClientMapMutex = sync.RWMutex{}

	// sid -> *uid
	TokenMap      = make(map[uuid.UUID]*uuid.UUID)
	TokenMapMutex = sync.RWMutex{}

	// sid -> *Hub
	NewHubs = make(map[uuid.UUID]*Hub)
)

type MsgType int

const (
	DEBUG MsgType = iota
	_
	ROOM_EVENT
	_
	PLAYLIST_EVENT
	_
	RESERVED
)

/*
type 0: debug
type 1: room event
type 3: playlist event
type 5: reserved
*/
type Message struct {
	MsgType  MsgType
	From     uuid.UUID
	To       uuid.UUID
	UID      string
	Username string
	Data     interface{} `json:"Data"`
}

type Hub struct {
	ID        uuid.UUID
	hubctx    context.Context
	hubcancel func()
	Host      *Client
	Clients   map[*Client]int // multiple host is allowed
	Playlist  *Playlist
	Player    *mq.WorkerPool

	// hub control channel
	Register   chan *Client
	Unregister chan *Client
	Destroy    chan struct{}

	// playlist control channel
	AddedSong chan struct{}
	NextSong  chan struct{}

	// message channel
	broadcast chan Message
	direct    chan Message
}

func CreateHub(id uuid.UUID) *Hub {
	clients := make(map[*Client]int)
	playlist := CreatePlaylist()
	mq, err := mq.NewWorkerPool(1, 1)
	if err != nil {
		slog.Error("task queue err", "err", err)
		return &Hub{}
	}

	// // the first client is the host by default
	// clients[client] = 7

	return &Hub{
		ID:       id,
		Host:     nil,
		Clients:  clients,
		Playlist: playlist,
		Player:   mq,

		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Destroy:    make(chan struct{}),

		AddedSong: make(chan struct{}),
		NextSong:  make(chan struct{}),

		broadcast: make(chan Message),
		direct:    make(chan Message),
	}
}

func (h *Hub) Run() {
	defer func() {
		close(h.Destroy)
		delete(HubMap, h.ID)
		h.Playlist.Clear()
		h.hubcancel()
	}()
	// start mq
	hubctx, hubshutdown := context.WithCancel(context.Background())
	mqctx := context.WithValue(hubctx, "name", h.ID.String())
	h.hubctx = hubctx
	h.hubcancel = hubshutdown
	go h.Player.Run(mqctx)

	for {
		select {
		case client := <-h.Register:
			h.Clients[client] = client.Permission
			// set host
			if h.Host == nil {
				h.Host = client
			}
		case client := <-h.Unregister:
			ClientMapMutex.Lock()
			TokenMapMutex.Lock()

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
				delete(ClientMap, client.ID)
				delete(TokenMap, client.Token)
				close(client.Send)
				// check if hub should be closed
				if len(h.Clients) == 0 {
					base64rid := base64.RawURLEncoding.EncodeToString(h.ID[:])
					slog.Debug("ws hub closed: no client in hub", "id", base64rid)

					// unlock mutex
					TokenMapMutex.Unlock()
					ClientMapMutex.Unlock()
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
			// unlock mutex
			TokenMapMutex.Unlock()
			ClientMapMutex.Unlock()
		case msg := <-h.broadcast:
			msgJson, err := json.Marshal(msg)
			if err != nil {
				slog.Error("json err", "err", err)
			}
			slog.Debug("ws msg", "uid", msg.UID, "msg", msg.Data)
			for client := range h.Clients {
				select {
				case client.Send <- []byte(msgJson):
				default:
					close(client.Send)
					delete(h.Clients, client)
				}
			}
		case cmd := <-h.Destroy:
			slog.Debug("hub received c4", "cmd", cmd)
			return
		case <-h.AddedSong:
			slog.Debug("hub check playlist")
			go h.enqueuedPlaylist()
		}
	}
}

func (h *Hub) enqueuedPlaylist() {
	ctx, cancel := context.WithTimeout(context.Background(), ytdlp.TIMEOUT_AUDIO)
	node, err := h.Playlist.Dequeue()
	if err != nil {
		slog.Error("hub dequeue err", "err", err)
	}
	req := ytdlp.RequestAudio{
		Ctx:      ctx,
		URL:      node.URL,
		Response: make(chan any),
	}

	go func() {
		defer cancel()

		// mq response
		select {
		case <-ctx.Done():
			slog.Debug("[hub] request audio done")
			return
		case res := <-req.Response:
			if accepted, ok := res.(bool); ok {
				slog.Debug("[hub] status", "status", accepted)
				if !accepted {
					return
				}
			}
		}

		// audio byte response
		select {
		case <-ctx.Done():
			slog.Debug("[hub] request audio done")
			return
		case res := <-req.Response:
			if audioBytes, ok := res.([]byte); ok {
				slog.Debug("wtf1")
				msg := Message{
					MsgType: DEBUG,
					Data:    audioBytes,
				}
				h.BroadcastMsg(msg)
				slog.Debug("wtf2")
			}
		}
	}()
	ytdlp.AudioDownloader.Submit(ctx, &req)
}

func (h *Hub) Timeout(sid *uuid.UUID) {
	select {
	case <-time.After(TIMEOUT_HUB):
		// close the hub if no one joined after some time
		base64rid := base64.RawURLEncoding.EncodeToString(h.ID[:])
		delete(NewHubs, *sid)
		if len(h.Clients) == 0 {
			slog.Debug("hub auto close", "rid", base64rid)
			select {
			case <-h.Destroy:
				// closed
				// slog.Debug("timeout destroy closed")
				return
			default:
			}
			h.Destroy <- struct{}{}
			// slog.Debug("timeout not blocking")
			return
		} else {
			// slog.Debug("keep running", "rid", base64rid)
			return
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
	// slog.Debug("broadcast start")
	if len(h.Clients) > 0 {
		select {
		case <-h.Destroy:
			// slog.Debug("broadcast destroy closed")
			return
		default:
		}
		h.broadcast <- msg
		// slog.Debug("broadcast not blocking")
		return
	}
	// slog.Debug("broadcast no client")
}
