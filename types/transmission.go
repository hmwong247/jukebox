package types

// NOTE: all id in UserSession are base64 encoded

type PublicRoom struct {
	RoomID     string
	Host       string
	Capacity   int
	RequirePin bool
}

type UserSession struct {
	Username string
	UserID   string
	RoomID   string
}
