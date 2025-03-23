package views

import "main/internal/room"

type RoomStatus struct {
	RoomID   string
	Host     string
	Capacity int
	UserList map[*room.Client]int
}
