package api

import (
	"encoding/base64"
	"log/slog"
	"main/core/corewebsocket"
	"main/types"
	"net/http"
	"strings"
	"text/template"

	"github.com/google/uuid"
)

var (
	// template cache
	// tmplHome template.Template

	// for websocket connect that maps session id to user profile
	entryMap = make(map[uuid.UUID]UserProfile)

	musicRooms = make(map[uuid.UUID]*types.RoomInfo)
)

type UserProfile struct {
	name string
	uid  uuid.UUID
	rid  uuid.UUID
}

func (userProfile *UserProfile) Index(s []UserProfile) int {
	for i, other := range s {
		if userProfile.uid == other.uid {
			return i
		}
	}
	return -1
}

// route: "GET /" forbidden
func HandleRoot(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "forbidden", http.StatusForbidden)
}

// route: "GET /home"
func HandleDefault(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("statics/index.html", "templates/forms/user_profile.html"))
	tmpl.Execute(w, nil)
}

// route: "GET /join"
func HandleJoin(w http.ResponseWriter, r *http.Request) {
	// joinRoomID := r.PathValue("rid")
	queryParam := r.URL.Query()
	qRID := strings.TrimSpace(queryParam.Get("rid"))
	slog.Debug("GET /join", "rid", qRID)
	if qRID == "" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	tmpl := template.Must(template.ParseFiles("statics/join.html", "templates/forms/user_profile.html"))
	tmpl.Execute(w, nil)

	// userUUIDBytes, err := base64.RawURLEncoding.DecodeString(joinUserID)
	// userUUID, err := uuid.FromBytes(userUUIDBytes)
	// if err != nil {
	// 	log.Printf("Invalid user UUID from client: %v\n", err)
	// 	return
	// }
	// roomUUIDBytes, err := base64.RawURLEncoding.DecodeString(joinRoomID)
	// roomUUID, err := uuid.FromBytes(roomUUIDBytes)
	// if err != nil {
	// 	log.Printf("Invalid room UUID from client: %v\n", err)
	// 	return
	// }
	// room, roomExists := musicRooms[roomUUID]
	// var currentRoom types.CurrentRoom
	// if roomExists {
	// 	user := types.User{
	// 		UserID:   userUUID,
	// 		UserName: joinUsername,
	// 	}
	// 	userExists := user.Index(room.UserList)
	// 	if userExists == -1 {
	// 		room.UserList = append(room.UserList, user)
	// 	}
	//
	// 	currentRoom = types.CurrentRoom{
	// 		RoomID:   joinRoomID, // no change, if exists
	// 		UserID:   joinUserID,
	// 		Username: joinUserID,
	// 		Host:     room.Host,
	// 		Capacity: len(room.UserList),
	// 		UserList: room.UserList,
	// 	}
	// } else {
	// 	currentRoom = types.CurrentRoom{
	// 		RoomID:   "n/a",
	// 		UserID:   joinUserID,
	// 		Username: joinUserID,
	// 		Host:     "n/a",
	// 		Capacity: 0,
	// 	}
	// }
	//
	// tmpl := template.Must(template.ParseGlob("templates/CurrentRoom.html"))
	// tmpl.ExecuteTemplate(w, "current_room", currentRoom)
}

// route: GET /lobby
func EnterLobby(w http.ResponseWriter, r *http.Request) {
	queryParam := r.URL.Query()
	qRoomID := strings.TrimSpace(queryParam.Get("rid"))
	decodebase64, err := base64.RawURLEncoding.DecodeString(qRoomID)
	roomID, err := uuid.FromBytes(decodebase64)
	if err != nil {
		slog.Info("Trying to enter lobby with invalid room UUID", "status", http.StatusForbidden)
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	// check roomID exists, check user exists in room
	hub, ok := corewebsocket.HubMap[roomID]
	if !ok {
		slog.Error("hub not found", "status", http.StatusInternalServerError, "rid", roomID.String())
		http.Error(w, "forbidden", http.StatusInternalServerError)
		return
	}
	tmpl := template.Must(template.ParseGlob("templates/CurrentRoom.html"))
	// encodedBase64URL := base64.RawURLEncoding.EncodeToString(newRoom.RoomID[:])
	session := types.CurrentRoom{
		RoomID: qRoomID,
		// Host:     room.Host,
		Capacity: len(hub.Clients),
		// UserList: room.UserList,
	}
	tmpl.ExecuteTemplate(w, "room_status", session)
}

// route: "GET /api/new-user"
func HandleNewUser(w http.ResponseWriter, r *http.Request) {
	userID := uuid.New()
	userIDByte := userID[:]
	encodedBase64 := base64.RawURLEncoding.EncodeToString(userIDByte)
	slog.Debug("GET /new-user", "base64", encodedBase64)
	w.Write([]byte(encodedBase64))
}

// route: "GET /api/create"
func HandleCreateRoom(w http.ResponseWriter, r *http.Request) {
	rid := uuid.New()
	hub := corewebsocket.NewHub(rid)
	corewebsocket.HubMap[rid] = hub
	go hub.Run()

	encodedRID := base64.RawURLEncoding.EncodeToString(rid[:])
	w.Write([]byte(encodedRID))
}

// route: POST /api/session
func HandleNewSession(w http.ResponseWriter, r *http.Request) {
	// return a session id for websocket identificaion
	pUsername := r.PostFormValue("cfg_username")
	pUID := r.PostFormValue("user_id")
	pRID := r.PostFormValue("room_id")
	// never trust the client
	decodedUID, err := base64.RawURLEncoding.DecodeString(pUID)
	uid, err := uuid.FromBytes(decodedUID)
	if err != nil {
		slog.Info("Invalid user UUID from client:", "status", http.StatusForbidden, "err", err)
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	decodedRID, err := base64.RawURLEncoding.DecodeString(pRID)
	rid, err := uuid.FromBytes(decodedRID)
	if err != nil {
		slog.Info("Invalid room UUID from client:", "status", http.StatusForbidden, "err", err)
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var name string = strings.TrimSpace(pUsername)
	if name == "" {
		name = "user"
	}

	sid := uuid.New()
	profile := UserProfile{
		name: name,
		uid:  uid,
		rid:  rid,
	}
	entryMap[sid] = profile
	encodedBase64URL := base64.RawURLEncoding.EncodeToString(sid[:])
	w.Write([]byte(encodedBase64URL))
}
