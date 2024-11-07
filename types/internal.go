package types

import "github.com/google/uuid"

type User struct {
	UserID   uuid.UUID
	UserName string
}

type RoomInfo struct {
	RoomID   uuid.UUID
	IsPublic bool
	Host     string
	Pin      string
	UserList []User
}
