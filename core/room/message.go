package room

import (
	"main/core/mq"
	"main/core/ytdlp"

	"github.com/google/uuid"
)

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
	_
	MSG_RESERVED
)

type Message struct {
	MsgType  MsgType
	UID      string      `json:",omitempty"`
	Username string      `json:",omitempty"`
	Data     interface{} `json:"Data"`
}

type RoomEventMessage struct {
	MsgType  MsgType
	UID      string
	Username string
	Event    string
}

type WSInfoJson struct {
	ID int
	ytdlp.InfoJson
}

type PlaylistEventMessage struct {
	MsgType  MsgType
	UID      string
	Username string
	Data     WSInfoJson
}

type DirectMessage struct {
	MsgType MsgType
	To      uuid.UUID
	Data    mq.TaskStatus `json:"Data"`
}
