package room

//
// import (
// 	"sync"
// 	"time"
//
// 	"github.com/google/uuid"
// )
//
// var (
// 	// rid -> *Room
// 	RoomMap = make(map[uuid.UUID]*Room)
//
// 	// uid -> *Client
// 	ClientMap      = make(map[uuid.UUID]*Client)
// 	ClientMapMutex = sync.RWMutex{}
//
// 	// sid -> *uid
// 	TokenMap      = make(map[uuid.UUID]*uuid.UUID)
// 	TokenMapMutex = sync.RWMutex{}
//
// 	// sid -> *Hub
// 	NewHubs = make(map[uuid.UUID]*Hub)
// )
//
// type Room struct {
// 	ID       uuid.UUID
// 	Host     *Client
// 	Hub      *Hub
// 	Playlist *Playlist
// }
//
// func CreateRoom(rid uuid.UUID) *Room {
// 	return &Room{
// 		ID:       rid,
// 		Hub:      CreateHub(rid),
// 		Playlist: CreatePlaylist(),
// 	}
// }
//
// func (room *Room) Timeout(sid *uuid.UUID) {
// 	select {
// 	case <-time.After(5 * time.Second):
// 		// close the hub if no one joined after some time
// 		rid := base64.RawURLEncoding.EncodeToString(h.ID[:])
// 		if len(room.Clients) == 0 {
// 			slog.Debug("auto close", "rid", rid)
// 			delete(NewHubs, *sid)
// 			room.Destroy <- 0
// 			return
// 		} else {
// 			slog.Debug("keep running", "rid", rid)
// 			return
// 		}
// 	}
// }
