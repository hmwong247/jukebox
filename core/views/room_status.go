package views

import "main/core/room"

type RoomStatus struct {
	RoomID   string
	Host     string
	Capacity int
	UserList map[*room.Client]int
}
