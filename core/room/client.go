package room

import (
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	// buffer size
	READSIZE  = 1024
	WRITESIZE = 1024

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
		slog.Debug("ws client: defer read", "rid", c.Hub.ID.String(), "uid", c.ID.String())
	}()

	c.Conn.SetReadLimit(READSIZE)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error { c.Conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	for {
		_, msgRead, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Error("ws client read error", "err", err)
			}
			return
		}
		// msg = bytes.TrimSpace(bytes.Replace(msg, "\n", " ", -1))

		msg := Message{
			MsgType:  1,
			UID:      c.ID.String(),
			Username: c.Name,
			Data:     string(msgRead),
		}
		go c.Hub.BroadcastMsg(&msg)
	}
}

func (c *Client) Write() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
		slog.Debug("ws client: defer write", "rid", c.Hub.ID.String(), "uid", c.ID.String())
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
				slog.Error("ws client write err NextWriter", "err", err)
				return
			}
			w.Write(msg)
			if err := w.Close(); err != nil {
				slog.Error("ws client write err w.Close", "err", err)
				return
			}
		case <-ticker.C:
			// ping message
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				slog.Error("ws client write err ping", "err", err)
				return
			}
		}
	}
}
