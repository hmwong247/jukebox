package room

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"main/core/mq"
	"main/core/ytdlp"
	"math"
	"net/http"
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
	broadcast   chan *Message
	roomEvt     chan *RoomEventMessage
	playlistEvt chan *PlaylistEventMessage
	direct      chan *DirectMessage
}

func CreateHub(id uuid.UUID) *Hub {
	clients := make(map[*Client]int)
	playlist := NewPlaylist()
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

		broadcast:   make(chan *Message),
		roomEvt:     make(chan *RoomEventMessage),
		playlistEvt: make(chan *PlaylistEventMessage),
		direct:      make(chan *DirectMessage),
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
				msg := RoomEventMessage{
					MsgType:  MSG_EVENT_ROOM,
					UID:      client.ID.String(),
					Username: client.Name,
					Event:    "left",
				}
				go h.RoomEvtMsg(&msg)

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
						msg := RoomEventMessage{
							MsgType:  MSG_EVENT_ROOM,
							UID:      h.Host.ID.String(),
							Username: h.Host.Name,
							Event:    "host",
						}
						go h.RoomEvtMsg(&msg)
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
			if msg.MsgType == MSG_DEBUG {
				slog.Debug("[hub] ws msg", "msg", string(msgJson))
			}
			for client := range h.Clients {
				select {
				case client.Send <- []byte(msgJson):
				default:
					close(client.Send)
					delete(h.Clients, client)
				}
			}
		case msg := <-h.roomEvt:
			// broadcast room event message
			msgJson, err := json.Marshal(msg)
			if err != nil {
				slog.Error("json err", "err", err)
			}
			if msg.MsgType == MSG_DEBUG {
				slog.Debug("[hub] ws msg", "msg", string(msgJson))
			}
			for client := range h.Clients {
				select {
				case client.Send <- []byte(msgJson):
				default:
					close(client.Send)
					delete(h.Clients, client)
				}
			}
		case msg := <-h.playlistEvt:
			// broadcast playlist event message
			msgJson, err := json.Marshal(msg)
			if err != nil {
				slog.Error("json err", "err", err)
			}
			if msg.MsgType == MSG_DEBUG {
				slog.Debug("[hub] ws msg", "msg", string(msgJson))
			}
			for client := range h.Clients {
				select {
				case client.Send <- []byte(msgJson):
				default:
					close(client.Send)
					delete(h.Clients, client)
				}
			}

		case msg := <-h.direct:
			if msg.To == uuid.Nil {
				slog.Error("[hub] uid is not set in direct message")
				continue
			}
			msgJson, err := json.Marshal(msg)
			if err != nil {
				slog.Error("json err", "err", err)
			}
			if msg.MsgType == MSG_DEBUG {
				slog.Debug("[hub] ws msg", "msg", string(msgJson))
			}
			ClientMapMutex.RLock()
			if client, ok := ClientMap[msg.To]; ok {
				select {
				case client.Send <- []byte(msgJson):
				default:
					close(client.Send)
					delete(h.Clients, client)
				}
			}
			ClientMapMutex.RUnlock()
		case <-h.Destroy:
			slog.Debug("[hub] recieved destroy")
			return
		case <-h.AddedSong:
			slog.Debug("hub check playlist")
			go h.enqueuedPlaylist()
		}
	}
}

func (h *Hub) enqueuedPlaylist() {
	ctx, cancel := context.WithTimeout(context.Background(), ytdlp.TIMEOUT_AUDIO)
	defer cancel()
	node, err := h.Playlist.Dequeue()
	if err != nil {
		slog.Error("[hub] dequeue playlist err", "err", err)
		return
	}
	req := ytdlp.RequestAudio{
		Ctx:   ctx,
		URL:   node.URL,
		ErrCh: make(chan error),
		FinCh: make(chan struct{}),
	}
	status, _ := ytdlp.AudioDownloader.Submit(ctx, &req)

	// mq response
	if status != http.StatusAccepted {
		slog.Info("[hub] failed to enqueue request", "request", req)
		return
	}

	// audio byte response
	select {
	case <-ctx.Done():
		slog.Info("[hub] request audio timeout")
		return
	case err := <-req.ErrCh:
		slog.Info("[hub] audio byte reponse err", "req", req, "err", err)
		return
	case <-req.FinCh:
		msg := Message{
			MsgType: MSG_EVENT_PLAYLIST,
			Data:    req.Response,
		}
		h.BroadcastMsg(&msg)
	}
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

func (h *Hub) BroadcastMsg(msg *Message) {
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

func (h *Hub) RoomEvtMsg(msg *RoomEventMessage) {
	if len(h.Clients) > 0 {
		select {
		case <-h.Destroy:
			return
		default:
		}
		h.roomEvt <- msg
		return
	}
}

func (h *Hub) PlaylistMsg(msg *PlaylistEventMessage) {
	if len(h.Clients) > 0 {
		select {
		case <-h.Destroy:
			return
		default:
		}
		h.playlistEvt <- msg
		return
	}
}

func (h *Hub) DirectMsg(msg *DirectMessage) {
	slog.Debug("direct message start")
	if len(h.Clients) > 0 {
		select {
		case <-h.Destroy:
			slog.Debug("direct message destroy closed")
			return
		default:
		}
		h.direct <- msg
		slog.Debug("direct message not blocking")
		return
	}
	slog.Debug("directMsg no client")
}
