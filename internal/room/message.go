package room

import (
	"encoding/json"
	"errors"
	"fmt"
	"main/internal/mq"
	"main/internal/ytdlp"

	"github.com/google/uuid"
)

type WSMessage interface {
	Json() ([]byte, error)
	Client() *Client
	DebugMode() bool
}

// MsgType is still needed for frontend
// type 0: debug
// type 1: room entry event
// type 3: playlist event
// type 5: reserved
type MsgType int

const (
	MSG_DEBUG MsgType = iota
	MSG_EVENT_ROOM
	_
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
