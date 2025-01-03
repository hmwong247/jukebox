package views

import (
	"main/core/corewebsocket"
)

type RoomStatus struct {
	RoomID   string
	Host     string
	Capacity int
	UserList map[*corewebsocket.Client]int
}
