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
	HUB_TIMEOUT = 10
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

/*
type 0: debug
type 1: room event
type 3: playlist event
type 5: reserved
*/
type Message struct {
	MsgType  int
	From     uuid.UUID
	To       uuid.UUID
	UID      string
	Username string
	Data     interface{} `json:"Data"`
}

type AddRequest struct {
	// Node     *MusicNode
	Client   *Client
	URL      string
	Response chan error
}

type DelRequest struct {
	// Node     *MusicNode
	Client   *Client
	NodeID   int
	Response chan error
}

type Hub struct {
	ID        uuid.UUID
	hubctx    context.Context
	hubcancel func()
	Host      *Client
	Clients   map[*Client]int // multiple host is allowed
	Playlist  *Playlist
	MQ        *mq.WorkerPool

	// hub control channel
	Register   chan *Client
	Unregister chan *Client
	Destroy    chan struct{}

	// playlist control channel
	AddSong    chan *AddRequest
	RemoveSong chan *DelRequest
	NextSong   chan struct{}

	// message channel
	broadcast chan Message
	direct    chan Message
}

func CreateHub(id uuid.UUID) *Hub {
	clients := make(map[*Client]int)
	playlist := CreatePlaylist()
	mq, err := mq.NewWorkerPool(1, 4)
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
		MQ:       mq,

		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Destroy:    make(chan struct{}),

		AddSong:    make(chan *AddRequest),
		RemoveSong: make(chan *DelRequest),
		NextSong:   make(chan struct{}),

		broadcast: make(chan Message),
		direct:    make(chan Message),
	}
}

func (h *Hub) Run() {
	defer func() {
		// close all channels
		// close(h.Register)
		// close(h.Unregister)
		//
		// close(h.AddSong)
		// close(h.RemoveSong)
		// close(h.NextSong)
		//
		// close(h.Broadcast)
		// close(h.DirectMessage)
		close(h.Destroy)
		delete(HubMap, h.ID)
		h.Playlist.Clear()
		h.hubcancel()
	}()
	// start mq
	hubctx, hubshutdown := context.WithCancel(context.Background())
	mqctx := context.WithValue(hubctx, "name", h.ID.String())
	h.hubctx = hubctx
	h.hubcancel = func() { hubshutdown() }
	go h.MQ.Run(mqctx)

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
		case req := <-h.AddSong:
			node := MusicNode{
				URL: req.URL,
			}
			if err := h.Playlist.Enqueue(&node); err != nil {
				// response with error
				req.Response <- err
			} else {
				// debug log
				h.Playlist.Traverse()
				req.Response <- nil
				// forward to worker
				// h.Worker.addSong <- req
			}
		}
	}
}

func (h *Hub) Timeout(sid *uuid.UUID) {
	select {
	case <-time.After(HUB_TIMEOUT * time.Second):
		// close the hub if no one joined after some time
		base64rid := base64.RawURLEncoding.EncodeToString(h.ID[:])
		delete(NewHubs, *sid)
		if len(h.Clients) == 0 {
			slog.Debug("auto close", "rid", base64rid)
			select {
			case <-h.Destroy:
				// closed
				slog.Debug("timeout destroy closed")
				return
			default:
			}
			h.Destroy <- struct{}{}
			slog.Debug("timeout not blocking")
			return
		} else {
			slog.Debug("keep running", "rid", base64rid)
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
	slog.Debug("broadcast start")
	if len(h.Clients) > 0 {
		select {
		case <-h.Destroy:
			slog.Debug("broadcast destroy closed")
			return
		default:
		}
		h.broadcast <- msg
		slog.Debug("broadcast not blocking")
		return
	}
	slog.Debug("broadcast no client")
}

func fetchInfoJson(node *MusicNode, client *Client) {
	// audioByteArr, err := ytdlp.CmdStart(pURL)
	// if err != nil {
	// 	http.Error(w, "", http.StatusBadRequest)
	// 	return
	// }
	// byteReader := bytes.NewReader(audioByteArr)
	// w.Write(audioByteArr)
	// http.ServeContent(w, r, "", time.Time{}, byteReader)

	infoJson, err := ytdlp.DownloadInfoJson(node.URL)
	if err != nil {
		slog.Error("info json err", "err", err)
		msg := Message{
			MsgType: 3,
			To:      client.ID,
			Data:    "failed",
		}
		go client.Hub.BroadcastMsg(msg)
		return
	}
	// infoJson.ID = node.ID
	node.InfoJson = infoJson

	msg := Message{
		MsgType:  3,
		UID:      client.ID.String(),
		Username: client.Name,
		Data:     infoJson,
	}
	go client.Hub.BroadcastMsg(msg)
}
