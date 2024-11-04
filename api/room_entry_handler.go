package api

import (
	"encoding/base64"
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

// route: "/" forbidden
func HandleRoot(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "forbidden", http.StatusForbidden)
}

// route: "/home"
func HandleDefault(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("index.html", "templates/PublicRoomsDisplay.html"))
	tmpl.Execute(w, musicRooms)
}

// @TODO: join with URL, client side base64 encoding
// route: "/new-user"
func HandleNewUser(w http.ResponseWriter, r *http.Request) {
	// userSession := UserSession{
	// 	RoomID:   "",
	// 	UserID:   uuid.NewString(),
	// 	Username: "",
	// }
	//
	// ret, err := json.Marshal(userSession)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	userID := uuid.New()
	userIDByte := userID[:]
	encodedBase64 := base64.RawURLEncoding.EncodeToString(userIDByte)
	log.Printf("uuid: %#v, base64: %#v\n", userID, encodedBase64)
	w.Write([]byte(encodedBase64))
}

// route: "/join"
func HandleJoin(w http.ResponseWriter, r *http.Request) {
	// if r.Method != http.MethodPost {
	// 	http.Error(w, "", http.StatusForbidden)
	// 	return
	// }
	idString := r.PathValue("id")
	log.Println("id: ", idString)

	joinUsername := strings.TrimSpace(r.PostFormValue("join_username"))
	joinRoomID := strings.TrimSpace(r.PostFormValue("join_room_id"))
	room, exists := musicRooms[joinRoomID]
	var returnSession types.UserSession
	if exists {
		returnSession = types.UserSession{
			RoomID:   room.RoomID,
			Username: joinUsername,
		}
	} else {
		returnSession = types.UserSession{
			RoomID:   "n/a",
			Username: "n/a",
		}
	}
	tmpl := template.Must(template.ParseGlob("templates/CurrentRoom.html"))
	tmpl.ExecuteTemplate(w, "current_room", returnSession)
}

// route: "/create-room"
func HandleCreateRoom(w http.ResponseWriter, r *http.Request) {
	// if r.Method != http.MethodPost {
	// 	http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
	// 	return
	// }

	cfgUsername := r.PostFormValue("cfg_username")
	cfgPrivate := r.PostFormValue("cfg_private")
	cfgPin := r.PostFormValue("cfg_pin")

	var host string = strings.TrimSpace(cfgUsername)
	if host == "" {
		host = "user"
	}
	var isPrivate bool
	if cfgPrivate == "on" {
		isPrivate = true
	} else {
		isPrivate = false
	}
	var customPin string = ""
	if strings.Trim(cfgPin, " ") != "" {
		customPin = strings.TrimSpace(cfgPin)
	}
	log.Printf("pin: %#v\n", customPin)
	newRoom := CreateMusicRoom(host, isPrivate, customPin)
	_, exists := musicRooms[newRoom.RoomID]
	if exists {
		http.Error(w, "UUID collided", http.StatusInternalServerError)
		log.Printf("uuid collided: %#v\n", newRoom.RoomID)
	} else {
		musicRooms[newRoom.RoomID] = newRoom
	}

	tmpl := template.Must(template.ParseGlob("templates/CurrentRoom.html"))
	session := types.UserSession{
		RoomID:   newRoom.RoomID,
		Username: newRoom.Host,
	}
	tmpl.ExecuteTemplate(w, "current_room", session)
	tmpl = template.Must(template.ParseFiles("templates/PublicRoomsDisplay.html"))
	tmpl.ExecuteTemplate(w, "oob-public_rooms_display", newRoom)
}

var (
	musicRooms = make(map[string]types.RoomInfo)
)

func CreateMusicRoom(host string, isPublic bool, customPin string) types.RoomInfo {
	uuid := uuid.NewString()
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
		RoomID:    uuid,
		IsPublic:  isPublic,
		Host:      host,
		Pin:       customPin,
		NumUser:   1,
		UsersList: []string{},
	}
	return ret
}

func Debug_data() {
	// debug
	for i := 0; i < 5; i++ {
		newRoom := CreateMusicRoom("A", false, "")
		_, exists := musicRooms[newRoom.RoomID]
		if exists {
			log.Printf("uuid collided: %#v\n", newRoom.RoomID)
		} else {
			musicRooms[newRoom.RoomID] = newRoom
		}
	}

	// debug: check for dupilcated pin
	pinMap := make(map[string]int)
	for _, item := range musicRooms {
		_, exists := pinMap[item.Pin]
		if exists {
			pinMap[item.Pin] += 1
		} else {
			pinMap[item.Pin] = 1
		}
	}
	log.Println("check pin collide")
	for key, val := range pinMap {
		fmt.Printf("key: %#v, item: %#v\n", key, val)
	}

}
