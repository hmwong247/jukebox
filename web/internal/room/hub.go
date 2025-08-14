package room

import (
	"context"
	"encoding/base64"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
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

// debug
func (h *Hub) B64ID() string {
	return base64.RawURLEncoding.EncodeToString(h.ID[:])
}

func (c *Client) B64ID() string {
	return base64.RawURLEncoding.EncodeToString(c.ID[:])
}

// Hub should only control what a websocket hub should do
// seperate the music streaming to the music player
type Hub struct {
	ID        uuid.UUID
	hubctx    context.Context
	hubcancel func()
	Host      *Client
	Clients   map[*Client]int // multiple host is allowed
	Player    *MusicPlayer

	// hub control channel
	Register   chan *Client
	Unregister chan *Client
	Destroy    chan struct{}

	// message channel
	broadcast chan WSMessage
	direct    chan WSMessage
	peer      chan WSMessage
}

func CreateHub(id uuid.UUID) *Hub {
	clients := make(map[*Client]int)
	// // the first client is the host by default
	// clients[client] = 7

	return &Hub{
		ID:      id,
		Host:    nil,
		Clients: clients,
		Player:  CreateMusicPlayer(),

		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Destroy:    make(chan struct{}),

		broadcast: make(chan WSMessage),
		direct:    make(chan WSMessage),
		peer:      make(chan WSMessage),
	}
}

func (h *Hub) Run() {
	defer func() {
		close(h.Destroy)
		delete(HubMap, h.ID)
		h.hubcancel()
	}()

	// start music player
	hubctx, hubshutdown := context.WithCancel(context.Background())
	mpctx := context.WithValue(hubctx, "name", h.ID.String())
	h.hubctx = hubctx
	h.hubcancel = hubshutdown
	go h.Player.Run(mpctx, h)

	for {
		select {
		case <-h.Destroy:
			log.Info().Msg("[hub] recieved destroy")
			return

		case client := <-h.Register:
			h.register(client)

		case client := <-h.Unregister:
			h.unregister(client)

		case msg := <-h.broadcast:
			msgJson, err := msg.Json()
			if err != nil {
				log.Error().Err(err).
					Str("sender", msg.Sender().ID.String()).
					Msg("[hub] Broadcast msg json encode error")
				continue
			}
			if msg.DebugMode() {
				log.Debug().
					Str("sender", msg.Sender().ID.String()).
					RawJSON("msg", msgJson).
					Msg("[hub] ws broadcast msg")
			}
			for client := range h.Clients {
				select {
				case client.Send <- msgJson:
				default:
					close(client.Send)
					delete(h.Clients, client)
				}
			}

		case msg := <-h.direct:
			msgJson, err := msg.Json()
			if err != nil {
				log.Error().Err(err).
					Str("sender", msg.Sender().ID.String()).
					Msg("[hub] Direct msg json encode error")
				continue
			}
			if msg.DebugMode() {
				log.Debug().
					Str("sender", msg.Sender().ID.String()).
					RawJSON("msg", msgJson).
					Msg("[hub] ws direct msg")
			}
			if client := msg.Reciever(); client != nil {
				select {
				case client.Send <- msgJson:
				default:
					close(client.Send)
					delete(h.Clients, client)
				}
			}

		case msg := <-h.peer:
			msgJson, err := msg.Json()
			if err != nil {
				log.Error().Err(err).
					Str("sender", msg.Sender().ID.String()).
					Msg("[hub] peer msg json encode error")
				continue
			}
			if msg.DebugMode() {
				log.Debug().
					Str("sender", msg.Sender().ID.String()).
					RawJSON("msg", msgJson).
					Msg("[hub] ws peer msg")
			}

			reciever := msg.Reciever()
			if reciever != nil {
				select {
				case reciever.Send <- msgJson:
				default:
					close(reciever.Send)
					delete(h.Clients, reciever)
				}
			} else {
				sender := msg.Sender()
				if sender == nil {
					continue
				}
				for client := range h.Clients {
					if client != sender {
						select {
						case client.Send <- msgJson:
						default:
							close(client.Send)
							delete(h.Clients, client)
						}
					}
				}
			}

			// end of select
		}
	}
}

// channel functions

func (h *Hub) register(client *Client) {
	h.Clients[client] = client.Permission
	// set host
	if h.Host == nil {
		h.Host = client
	}
}

func (h *Hub) unregister(client *Client) {
	ClientMapMutex.Lock()
	TokenMapMutex.Lock()

	if _, ok := h.Clients[client]; ok {
		// broadcast leave notification
		msg := BroadcastMessage[Event]{
			MsgType:  MSG_EVENT_ROOM,
			UID:      client.ID.String(),
			Username: client.Name,
			Data:     "left",
		}
		go h.BroadcastMsg(&msg)

		// clean up
		delete(h.Clients, client)
		delete(ClientMap, client.ID)
		delete(TokenMap, client.Token)
		close(client.Send)
		// check if hub should be closed
		if len(h.Clients) == 0 {
			go func() {
				select {
				case <-h.Destroy:
					// closed
					log.Debug().Msg("[hub] timeout destroy closed")
					return
				default:
				}
				h.Destroy <- struct{}{}
			}()
			log.Info().
				Str("rid", h.B64ID()).
				Msg("[hub] ws hub closed: no client in hub")

			// unlock mutex
			TokenMapMutex.Unlock()
			ClientMapMutex.Unlock()
			return
		} else {
			// check host transfer
			if h.Host.ID == client.ID {
				h.Host = h.NextHost()
				msg := BroadcastMessage[Event]{
					MsgType:  MSG_EVENT_ROOM,
					UID:      h.Host.ID.String(),
					Username: h.Host.Name,
					Data:     "host",
				}
				go h.BroadcastMsg(&msg)
			}
		}
	}

	// unlock mutex
	TokenMapMutex.Unlock()
	ClientMapMutex.Unlock()
}

func (h *Hub) Timeout(sid *uuid.UUID) {
	select {
	case <-time.After(TIMEOUT_HUB):
		// close the hub if no one joined after some time
		delete(NewHubs, *sid)
		if len(h.Clients) == 0 {
			select {
			case <-h.Destroy:
				// closed
				log.Debug().Msg("[hub] timeout destroy closed")
				return
			default:
			}
			h.Destroy <- struct{}{}
			return
		} else {
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

func (h *Hub) BroadcastMsg(msg WSMessage) {
	// slog.Debug("broadcast start")
	if len(h.Clients) > 0 {
		select {
		case <-h.Destroy:
			log.Debug().Msg("[hub] broadcast destroy closed")
			return
		default:
		}
		h.broadcast <- msg
		// slog.Debug("broadcast not blocking")
		return
	}
	// slog.Debug("broadcast no client")
}

func (h *Hub) DirectMsg(msg WSMessage) {
	if len(h.Clients) > 0 {
		select {
		case <-h.Destroy:
			log.Debug().Msg("[hub] direct message destroy closed")
			return
		default:
		}
		h.direct <- msg
		return
	}
}

func (h *Hub) SignalMsg(msg WSMessage) {
	if len(h.Clients) > 0 {
		select {
		case <-h.Destroy:
			log.Debug().Msg("[hub] signal message destroy closed")
			return
		default:
		}
		h.peer <- msg
		return
	}
}
