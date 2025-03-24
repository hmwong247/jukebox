package room

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"main/internal/mq"
	"main/internal/ytdlp"

	"github.com/google/uuid"
)

// All message sent via the websocket hub MUST implement this interface
type WSMessage interface {
	Json() ([]byte, error)
	// Client() *Client
	Sender() *Client
	Reciever() *Client
	DebugMode() bool
}

// MsgType is still needed for frontend
type MsgType int

const (
	MSG_DEBUG MsgType = iota
	MSG_EVENT_ROOM
	MSG_EVENT_PEER
	MSG_EVENT_PLAYLIST
	MSG_EVENT_PLAYER
	MSG_RESERVED
)

// fallback message struct
type Message struct {
	MsgType  MsgType
	UID      string      `json:",omitempty"`
	Username string      `json:",omitempty"`
	Data     interface{} `json:"Data"`
}

type Event string
type BMData interface {
	Event | WSInfoJson
}

type WSInfoJson struct {
	ID             int
	Cmd            string
	MovedTo        int `json:",omitempty"`
	ytdlp.InfoJson `json:",omitempty"`
}

type BroadcastMessage[T BMData] struct {
	MsgType  MsgType
	UID      string
	Username string
	Data     T
}

func (bm *BroadcastMessage[T]) Json() ([]byte, error) {
	msgJson, err := json.Marshal(bm)
	if err != nil {
		errStr := fmt.Sprintf("json error, %v", err)
		newErr := errors.New(errStr)
		return nil, newErr
	}

	return msgJson, nil
}

func (bm *BroadcastMessage[T]) Sender() *Client {
	if bm.UID != uuid.Nil.String() {
		uid, err := uuid.Parse(bm.UID)
		if err != nil {
			slog.Error("Invalid uuid from peer message", "uuid", bm.UID)
			return nil
		}
		if c, ok := ClientMap[uid]; ok {
			return c
		}
	}
	return nil
}

func (bm *BroadcastMessage[T]) Reciever() *Client {
	return nil
}

func (bm *BroadcastMessage[T]) Client() *Client {
	return nil
}

func (bm *BroadcastMessage[T]) DebugMode() bool {
	if bm.MsgType == MSG_DEBUG {
		return true
	}
	return false
}

type DMData interface {
	[]byte | mq.TaskStatus | MPStatus
}

type DirectMessage[T DMData] struct {
	MsgType MsgType
	To      uuid.UUID `json:"-"`
	Data    T
}

func (dm *DirectMessage[T]) Json() ([]byte, error) {
	if dm.To == uuid.Nil {
		return nil, errors.New("uid is not set in direct message")
	}
	msgJson, err := json.Marshal(dm)
	if err != nil {
		errStr := fmt.Sprintf("json error, %v", err)
		newErr := errors.New(errStr)
		return nil, newErr
	}

	return msgJson, nil
}

func (dm *DirectMessage[T]) Sender() *Client {
	return nil
}

func (dm *DirectMessage[T]) Reciever() *Client {
	ClientMapMutex.RLock()
	defer ClientMapMutex.RUnlock()
	if client, ok := ClientMap[dm.To]; ok {
		return client
	}

	return nil
}

func (dm *DirectMessage[T]) Client() *Client {
	ClientMapMutex.RLock()
	defer ClientMapMutex.RUnlock()
	if client, ok := ClientMap[dm.To]; ok {
		return client
	}

	return nil
}

func (dm *DirectMessage[T]) DebugMode() bool {
	if dm.MsgType == MSG_DEBUG {
		return true
	}
	return false
}

type PMData interface {
	string
}

type PeerMessage[T PMData] struct {
	MsgType  MsgType
	UID      string
	Username string
	Data     T
}

func (pm *PeerMessage[T]) Json() ([]byte, error) {
	msgJson, err := json.Marshal(pm)
	if err != nil {
		errStr := fmt.Sprintf("json error, %v", err)
		newErr := errors.New(errStr)
		return nil, newErr
	}

	return msgJson, nil
}

func (pm *PeerMessage[T]) Sender() *Client {
	if pm.UID != uuid.Nil.String() {
		uid, err := uuid.Parse(pm.UID)
		if err != nil {
			slog.Error("Invalid sender uuid from peer message", "uuid", pm.UID)
			return nil
		}
		if c, ok := ClientMap[uid]; ok {
			return c
		}
	}
	return nil
}

func (pm *PeerMessage[T]) Reciever() *Client {
	return nil
}

func (pm *PeerMessage[T]) DebugMode() bool {
	if pm.MsgType == MSG_DEBUG {
		return true
	}
	return false
}

type PeerDirectMessage[T PMData] struct {
	MsgType  MsgType
	UID      string
	Username string
	To       string
	Data     T
}

func (pm *PeerDirectMessage[T]) Json() ([]byte, error) {
	msgJson, err := json.Marshal(pm)
	if err != nil {
		errStr := fmt.Sprintf("json error, %v", err)
		newErr := errors.New(errStr)
		return nil, newErr
	}

	return msgJson, nil
}

func (pm *PeerDirectMessage[T]) Sender() *Client {
	if pm.UID != uuid.Nil.String() {
		uid, err := uuid.Parse(pm.UID)
		if err != nil {
			slog.Error("Invalid sender uuid from peer direct message", "uuid", pm.UID)
			return nil
		}
		if c, ok := ClientMap[uid]; ok {
			return c
		}
	}
	return nil
}

func (pm *PeerDirectMessage[T]) Reciever() *Client {
	if pm.To != uuid.Nil.String() {
		uid, err := uuid.Parse(pm.To)
		if err != nil {
			slog.Error("Invalid reciever uuid from peer direct message", "uuid", pm.To)
			return nil
		}
		if c, ok := ClientMap[uid]; ok {
			return c
		}
	}
	return nil
}

func (pm *PeerDirectMessage[T]) DebugMode() bool {
	if pm.MsgType == MSG_DEBUG {
		return true
	}
	return false
}
