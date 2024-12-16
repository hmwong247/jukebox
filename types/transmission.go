package types

// NOTE: all id are base64 encoded for tranmission

type PublicRoom struct {
	RoomID   string
	Host     string
	Capacity int
	IsPublic bool
}

// type UserSession struct {
// 	Username string
// 	UserID   string
// 	RoomID   string
// }

// @TODO: stripe out UUID bytes/encode base64 from UserList
type CurrentRoom struct {
	RoomID   string
	UserID   string
	Username string
	Host     string
	Capacity int
	UserList []User
}
