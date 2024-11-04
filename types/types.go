package types

type RoomInfo struct {
	RoomID    string
	IsPublic  bool
	Host      string
	Pin       string
	NumUser   int
	UsersList []string
}

type UserSession struct {
	UserID   string
	Username string
	RoomID   string
}
