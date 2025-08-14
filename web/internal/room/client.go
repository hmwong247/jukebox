package room

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

const (
	// buffer size
	READSIZE  = 1024 * 8
	WRITESIZE = 1024 * 8

	// ping pong message time
	writeWait = 10 * time.Second
	pongWait  = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10
)

var (
	Upgrader = websocket.Upgrader{
		ReadBufferSize:  READSIZE,
		WriteBufferSize: WRITESIZE,
	}
)

/*
permission: allowed values are 1,3,7
1 = guest
3 = trusted
7 = host
*/
type Client struct {
	Conn          *websocket.Conn
	Hub           *Hub
	ID            uuid.UUID
	Token         uuid.UUID
	Name          string
	Permission    int
	Send          chan []byte
	JoinUnixMilli int64
}

func (c *Client) Read() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
		log.Debug().
			Str("uid", c.ID.String()).
			Str("rid", c.Hub.B64ID()).
			Msg("[ws] client defer write")
	}()

	c.Conn.SetReadLimit(READSIZE)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error { c.Conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	for {
		_, msgRead, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Error().Err(err).Str("uid", c.ID.String()).Msg("[ws] Client unexpected read error")
			}
			log.Warn().Err(err).Str("uid", c.ID.String()).Msg("[ws] Client read error")
			return
		}
		// msg = bytes.TrimSpace(bytes.Replace(msg, "\n", " ", -1))
		// msgStr := string(msgRead)

		var rawMsg RawPeerSignalMessage
		err = nil
		err = json.Unmarshal(msgRead, &rawMsg)
		if err != nil {
			log.Warn().Err(err).Str("uid", c.ID.String()).Msg("[ws] Client json decode error")
			continue
		}
		if rawMsg.To != uuid.Nil.String() {
			msg := PeerDirectMessage[string]{
				MsgType:  MSG_EVENT_PEER,
				UID:      c.ID.String(),
				Username: c.Name,
				To:       rawMsg.To,
				Data:     string(msgRead),
			}
			go c.Hub.SignalMsg(&msg)
		} else {
			msg := PeerMessage[string]{
				MsgType:  MSG_EVENT_PEER,
				UID:      c.ID.String(),
				Username: c.Name,
				Data:     string(msgRead),
			}
			go c.Hub.SignalMsg(&msg)
		}

		// msg := PeerMessage[string]{
		// 	MsgType:  MSG_EVENT_PEER,
		// 	UID:      c.ID.String(),
		// 	Username: c.Name,
		// 	Data:     string(msgRead),
		// }
		// go c.Hub.SignalMsg(&msg)
	}
}

func (c *Client) Write() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
		log.Debug().
			Str("uid", c.ID.String()).
			Str("rid", c.Hub.B64ID()).
			Msg("[ws] client defer write")
	}()

	for {
		select {
		case msg, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// channel is closed by the hub
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				log.Error().Err(err).Str("uid", c.ID.String()).Msg("[ws] Client NextWriter error")
				return
			}
			w.Write(msg)
			if err := w.Close(); err != nil {
				log.Warn().Err(err).Str("uid", c.ID.String()).Msg("[ws] Client writer close error")
				return
			}
		case <-ticker.C:
			// ping message
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Warn().Err(err).Str("uid", c.ID.String()).Msg("[ws] Client ping error")
				return
			}
		}
	}
}

// helper functions to control music player
func (c *Client) SignalMPAdd() {
	c.Hub.Player.AddedSong <- struct{}{}
}

func (c *Client) SignalMPNext() {
	c.Hub.Player.NextSong <- struct{}{}
}

func (c *Client) SignalMPPreload() {
	c.Hub.Player.Preload <- struct{}{}
}

type RawPeerSignalMessage struct {
	To   string
	Data interface{} // don't care
}
