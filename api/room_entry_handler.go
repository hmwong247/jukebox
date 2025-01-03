package api

import (
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"main/core/corewebsocket"
	"main/core/views"
	"net/http"
	"strings"
	"text/template"

	"github.com/google/uuid"
)

var (
	// template cache
	// tmplHome template.Template

	// sid -> UserProfile
	entryProfiles = make(map[uuid.UUID]*UserProfile)
)

type UserProfile struct {
	name string
	uid  uuid.UUID
	rid  uuid.UUID
	sid  uuid.UUID
}

func (userProfile *UserProfile) Index(s []UserProfile) int {
	for i, other := range s {
		if userProfile.uid == other.uid {
			return i
		}
	}
	return -1
}

func decodeQueryID(r *http.Request, key string) (uuid.UUID, error) {
	queryParam := r.URL.Query()
	qID := strings.TrimSpace(queryParam.Get(key))
	decodedID, err := base64.RawURLEncoding.DecodeString(qID)
	id, err := uuid.FromBytes(decodedID)
	if err != nil {
		slog.Info("invalid UUID", "qID", qID)
		return id, err
	}
	slog.Debug("decode uuid", "key", key, "qID", qID)

	return id, nil
}

/*
	Pages
*/

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
	rid, err := decodeQueryID(r, "rid")
	if err != nil {
		http.Error(w, "", http.StatusForbidden)
		return
	}
	if _, ok := corewebsocket.HubMap[rid]; !ok {
		slog.Info("hub not found", "rid", rid.String())
		http.Error(w, "", http.StatusForbidden)
		return
	}

	tmpl := template.Must(template.ParseFiles("statics/join.html", "templates/forms/user_profile.html"))
	tmpl.Execute(w, nil)
}

// route: GET /lobby?sid=
func EnterLobby(w http.ResponseWriter, r *http.Request) {
	sid, err := decodeQueryID(r, "sid")
	if err != nil {
		slog.Info("Trying to enter lobby with invalid session UUID", "status", http.StatusForbidden)
		http.Error(w, "", http.StatusForbidden)
		return
	}
	client, ok := corewebsocket.ClientMap[sid]
	if !ok {
		slog.Info("client not found", "status", http.StatusInternalServerError, "sid", sid.String())
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	// check roomID exists, check user exists in room
	hub, ok := corewebsocket.HubMap[client.Hub.ID]
	if !ok {
		slog.Error("hub not found", "status", http.StatusInternalServerError, "rid", sid.String())
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
	base64RID := base64.RawURLEncoding.EncodeToString(hub.ID[:])
	tmpl := template.Must(template.ParseGlob("templates/CurrentRoom.html"))
	session := views.RoomStatus{
		RoomID:   base64RID,
		Host:     hub.Host.Name,
		Capacity: len(hub.Clients),
		UserList: hub.Clients,
	}
	tmpl.ExecuteTemplate(w, "room_status", session)
}

/*
	API
*/

// route: "GET /api/new-user"
func HandleNewUser(w http.ResponseWriter, r *http.Request) {
	userID := uuid.New()
	userIDByte := userID[:]
	encodedBase64 := base64.RawURLEncoding.EncodeToString(userIDByte)
	slog.Debug("GET /new-user", "base64", encodedBase64)
	w.Write([]byte(encodedBase64))
}

// route: "GET /api/create?sid"
func HandleCreateRoom(w http.ResponseWriter, r *http.Request) {
	sid, err := decodeQueryID(r, "sid")
	if err != nil {
		http.Error(w, "", http.StatusForbidden)
		return
	}

	// valid data
	rid := uuid.New()
	userProfile, ok := entryProfiles[sid]
	if !ok {
		slog.Error("user profile not found", "status", http.StatusForbidden, "sid", sid.String())
		return
	}
	userProfile.rid = rid

	hub := corewebsocket.CreateHub(rid)
	corewebsocket.HubMap[rid] = hub
	corewebsocket.NewHubs[sid] = hub

	base64RID := base64.RawURLEncoding.EncodeToString(rid[:])
	w.Write([]byte(base64RID))
}

// route: POST /api/session
func HandleNewSession(w http.ResponseWriter, r *http.Request) {
	// return a session id
	pUsername := r.PostFormValue("cfg_username")
	pUID := r.PostFormValue("user_id")

	// never trust the client
	decodedUID, err := base64.RawURLEncoding.DecodeString(pUID)
	uid, err := uuid.FromBytes(decodedUID)
	if err != nil {
		slog.Info("Invalid user UUID from client:", "status", http.StatusForbidden, "err", err)
		http.Error(w, "", http.StatusForbidden)
		return
	}
	var name string = strings.TrimSpace(pUsername)

	var rid uuid.UUID
	pRID := r.PostFormValue("room_id")
	if len(pRID) > 0 {
		decodedRID, err := base64.RawURLEncoding.DecodeString(pRID)
		_rid, err := uuid.FromBytes(decodedRID)
		rid = _rid
		if err != nil {
			slog.Info("Invalid room UUID from client:", "status", http.StatusForbidden, "err", err)
			http.Error(w, "", http.StatusForbidden)
			return
		}
	}

	// valid data, cache the user profile
	sid := uuid.New()
	profile := &UserProfile{
		name: name,
		uid:  uid,
		sid:  sid,
		rid:  rid,
	}
	entryProfiles[sid] = profile

	base64SID := base64.RawURLEncoding.EncodeToString(sid[:])
	w.Write([]byte(base64SID))
}

// route: "GET /api/users?sid="
func UserList(w http.ResponseWriter, r *http.Request) {
	sid, err := decodeQueryID(r, "sid")
	if err != nil {
		http.Error(w, "", http.StatusForbidden)
		return
	}
	client, ok := corewebsocket.ClientMap[sid]
	if !ok {
		slog.Info("client not found from sid", "sid", sid.String())
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	userlist := make(map[string]string)
	for c := range client.Hub.Clients {
		userlist[c.ID.String()] = c.Name
	}

	json, err := json.Marshal(userlist)
	w.Write(json)
}
