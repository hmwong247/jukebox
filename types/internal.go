package types

import "github.com/google/uuid"

type User struct {
	UserID   uuid.UUID
	UserName string
}

func (user *User) Index(s []User) int {
	for i, other := range s {
		if user.UserID == other.UserID {
			return i
		}
	}
	return -1
}

type RoomInfo struct {
	RoomID   uuid.UUID
	IsPublic bool
	Host     string
	Pin      string
	UserList []User
}
