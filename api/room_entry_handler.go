package api

import (
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"main/types"
	"math/rand/v2"
	"net/http"
	"strings"
	"text/template"

	// switch to ULID
	"github.com/google/uuid"
)

// route: "GET /" forbidden
func HandleRoot(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "forbidden", http.StatusForbidden)
}

// route: "GET /home"
func HandleDefault(w http.ResponseWriter, r *http.Request) {
	publicRooms := FilterPublicRooms(musicRooms)
	session := types.CurrentRoom{
		Username: "n/a",
		RoomID:   "n/a",
	}
	tmplData := types.TmplDataHome{
		CurrentRoom: session,
		PublicRooms: publicRooms,
	}

	tmpl := template.Must(template.ParseFiles("index.html", "templates/PublicRoomsDisplay.html", "templates/CurrentRoom.html"))
	tmpl.Execute(w, tmplData)
}

// route: "GET /new-user"
func HandleNewUser(w http.ResponseWriter, r *http.Request) {
	userID := uuid.New()
	userIDByte := userID[:]
	encodedBase64 := base64.RawURLEncoding.EncodeToString(userIDByte)
	log.Printf("GET /new-user, uuid: %#v, rawURL: %#v\n", userID.String(), encodedBase64)
	w.Write([]byte(encodedBase64))
}

// route: "GET /join"
func HandleJoin(w http.ResponseWriter, r *http.Request) {
	joinRoomID := r.PathValue("id")
	log.Println("GET /join, id: ", joinRoomID)
	if joinRoomID == "" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	queryParam := r.URL.Query()
	joinUsername := strings.TrimSpace(queryParam.Get("join_username"))
	joinUserID := strings.TrimSpace(queryParam.Get("join_userid"))

	if joinUserID == "" {
		fmt.Println("bootstrap client")
		http.ServeFile(w, r, "./bootstrap.html")
		return
	}
	userUUIDBytes, err := base64.RawURLEncoding.DecodeString(joinUserID)
	userUUID, err := uuid.FromBytes(userUUIDBytes)
	if err != nil {
		log.Printf("Invalid user UUID from client: %v\n", err)
		return
	}
	roomUUIDBytes, err := base64.RawURLEncoding.DecodeString(joinRoomID)
	roomUUID, err := uuid.FromBytes(roomUUIDBytes)
	if err != nil {
		log.Printf("Invalid room UUID from client: %v\n", err)
		return
	}
	room, roomExists := musicRooms[roomUUID]
	var currentRoom types.CurrentRoom
	if roomExists {
		user := types.User{
			UserID:   userUUID,
			UserName: joinUsername,
		}
		userExists := user.Index(room.UserList)
		if userExists == -1 {
			room.UserList = append(room.UserList, user)
		}

		currentRoom = types.CurrentRoom{
			RoomID:   joinRoomID, // no change, if exists
			UserID:   joinUserID,
			Username: joinUserID,
			Host:     room.Host,
			Capacity: len(room.UserList),
			UserList: room.UserList,
		}
	} else {
		currentRoom = types.CurrentRoom{
			RoomID:   "n/a",
			UserID:   joinUserID,
			Username: joinUserID,
			Host:     "n/a",
			Capacity: 0,
		}
	}
	publicRooms = FilterPublicRooms(musicRooms)

	tmpl := template.Must(template.ParseGlob("templates/CurrentRoom.html"))
	tmpl.ExecuteTemplate(w, "current_room", currentRoom)
}

// route: "GET /current-room/{id}"
func HandleGetCurrentRoom(w http.ResponseWriter, r *http.Request) {

}

// route: "POST /create-room"
func HandleCreateRoom(w http.ResponseWriter, r *http.Request) {
	cfgUsername := r.PostFormValue("cfg_username")
	cfgPrivate := r.PostFormValue("cfg_private")
	cfgPin := r.PostFormValue("cfg_pin")
	postUserID := r.PostFormValue("user_id")
	// never trust the client
	decodedBase64, err := base64.RawURLEncoding.DecodeString(postUserID)
	postUserUUID, err := uuid.FromBytes(decodedBase64)
	if err != nil {
		log.Printf("Invalid user UUID from client: %v\n", err)
		return
	}

	var host string = strings.TrimSpace(cfgUsername)
	if host == "" {
		host = "user"
	}
	var isPublic bool = true
	if cfgPrivate == "on" {
		isPublic = false
	}
	var customPin string = ""
	if strings.Trim(cfgPin, " ") != "" {
		customPin = strings.TrimSpace(cfgPin)
	}
	hostUser := types.User{
		UserID:   postUserUUID,
		UserName: host,
	}
	newRoom := CreateMusicRoom(hostUser, isPublic, customPin)
	_, exists := musicRooms[newRoom.RoomID]
	if exists {
		http.Error(w, "UUID collided", http.StatusInternalServerError)
		log.Printf("uuid collided: %#v\n", newRoom.RoomID)
		return
	} else {
		musicRooms[newRoom.RoomID] = &newRoom
		publicRooms = FilterPublicRooms(musicRooms)
	}

	tmpl := template.Must(template.ParseGlob("templates/CurrentRoom.html"))
	encodedBase64URL := base64.RawURLEncoding.EncodeToString(newRoom.RoomID[:])
	session := types.CurrentRoom{
		RoomID:   encodedBase64URL,
		UserID:   postUserID, // valided from the beginning
		Username: newRoom.Host,
		Host:     newRoom.Host,
		Capacity: len(newRoom.UserList),
		UserList: newRoom.UserList,
	}
	tmpl.ExecuteTemplate(w, "current_room", session)

	// @TODO: maybe put PublicRoom in a map?
	if newRoom.IsPublic {
		pubRoom, err := findPublicRoom(newRoom.RoomID, publicRooms)
		if err != nil {
			log.Println(err)
			return
		}
		tmpl = template.Must(template.ParseFiles("templates/PublicRoomsDisplay.html"))
		tmpl.ExecuteTemplate(w, "oob-public_rooms_display", pubRoom)
	}
}

/*
	helper functions
	@TODO: move these functions to core module
*/

var (
	musicRooms = make(map[uuid.UUID]*types.RoomInfo)

	publicRooms = make([]types.PublicRoom, 0)
)

func CreateMusicRoom(hostUser types.User, isPublic bool, customPin string) types.RoomInfo {
	uuid := uuid.New()
	if !isPublic && customPin == "" {
		const pinLen = 6
		pinPool := "0123456789"
		pinByte := make([]byte, pinLen)
		pinByte[0] = pinPool[rand.IntN(len(pinPool)-1)+1]
		for i := 1; i < pinLen; i++ {
			pinByte[i] = pinPool[rand.IntN(len(pinPool))]
		}
		customPin = string(pinByte)
	}

	ret := types.RoomInfo{
		RoomID:   uuid,
		IsPublic: isPublic,
		Host:     hostUser.UserName,
		Pin:      customPin,
		UserList: []types.User{hostUser},
	}
	return ret
}

func Debug_data() {
	// debug
	hostUser := types.User{UserName: "A"}
	for i := 0; i < 5; i++ {
		newRoom := CreateMusicRoom(hostUser, true, "")
		_, exists := musicRooms[newRoom.RoomID]
		if exists {
			log.Printf("uuid collided: %#v\n", newRoom.RoomID)
		} else {
			musicRooms[newRoom.RoomID] = &newRoom
		}
	}
	publicRooms = FilterPublicRooms(musicRooms)
}

func FilterPublicRooms(rooms map[uuid.UUID]*types.RoomInfo) []types.PublicRoom {
	ret := make([]types.PublicRoom, 0)
	for _, roomInfo := range rooms {
		if roomInfo.IsPublic {
			base64RoomID := base64.URLEncoding.EncodeToString(roomInfo.RoomID[:])
			requirePin := true
			if roomInfo.Pin == "" {
				requirePin = false
			}
			publicRoom := types.PublicRoom{
				RoomID:     base64RoomID,
				Host:       roomInfo.Host,
				Capacity:   len(roomInfo.UserList),
				RequirePin: requirePin,
			}
			ret = append(ret, publicRoom)
		}
	}

	// for _, room := range ret {
	// 	fmt.Printf("publicRoom: %#v\n", room)
	// }

	return ret
}

func findPublicRoom(roomID uuid.UUID, rooms []types.PublicRoom) (types.PublicRoom, error) {
	base64UUID := base64.URLEncoding.EncodeToString(roomID[:])
	for _, room := range rooms {
		if base64UUID == room.RoomID {
			return room, nil
		}
	}
	return types.PublicRoom{}, errors.New("public room not found")
}
