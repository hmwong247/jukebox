package api

import (
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"main/internal/room"
	"main/internal/views"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/google/uuid"
)

const (
	TIMEOUT_API_ENTRY = 10 * time.Second
)

var (
	// template cache
	// tmplHome template.Template

	// sid -> *UserProfile
	entryProfiles = make(map[uuid.UUID]*UserProfile)

	// uid -> *sid
	entryToken = make(map[uuid.UUID]*uuid.UUID)
)

type UserProfile struct {
	name string
	uid  uuid.UUID
	rid  uuid.UUID
	sid  uuid.UUID
}

func (userProfile *UserProfile) timeout() {
	select {
	case <-time.After(TIMEOUT_API_ENTRY):
		delete(entryProfiles, userProfile.sid)
		delete(entryToken, userProfile.uid)
		// slog.Debug("delete user profile")
		return
	}
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
	// slog.Debug("decode uuid", "key", key, "qID", qID)

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
		http.Error(w, "", http.StatusBadRequest)
		return
	}
	if _, ok := room.HubMap[rid]; !ok {
		slog.Info("hub not found", "rid", rid.String())
		http.Error(w, "", http.StatusForbidden)
		return
	}

	tmpl := template.Must(template.ParseFiles("statics/join.html", "templates/forms/user_profile.html"))
	tmpl.Execute(w, nil)
}

// route: GET /lobby?sid=
func EnterLobby(w http.ResponseWriter, r *http.Request) {
	room.ClientMapMutex.RLock()
	room.TokenMapMutex.RLock()
	defer func() {
		room.TokenMapMutex.RUnlock()
		room.ClientMapMutex.RUnlock()
	}()

	sid, err := decodeQueryID(r, "sid")
	if err != nil {
		slog.Info("Trying to enter lobby with invalid session UUID", "status", http.StatusForbidden)
		http.Error(w, "", http.StatusBadRequest)
		return
	}
	var client *room.Client
	if uid, ok := room.TokenMap[sid]; ok {
		if _client, ok := room.ClientMap[*uid]; ok {
			client = _client
		} else {
			slog.Info("client not found", "status", http.StatusInternalServerError, "uid", _client.ID.String())
			http.Error(w, "", http.StatusForbidden)
			return
		}
	} else {
		slog.Info("token not found", "status", http.StatusForbidden, "sid", sid.String())
		http.Error(w, "", http.StatusForbidden)
		return
	}

	// render room status
	base64RID := base64.RawURLEncoding.EncodeToString(client.Hub.ID[:])
	tmpl := template.Must(template.ParseGlob("templates/CurrentRoom.html"))
	session := views.RoomStatus{
		RoomID:   base64RID,
		Host:     client.Hub.Host.Name,
		Capacity: len(client.Hub.Clients),
		UserList: client.Hub.Clients,
	}
	tmpl.ExecuteTemplate(w, "room_status", session)

	// render music player
	tmpl = template.Must(template.ParseFiles("templates/MusicPlayer.html"))
	tmpl.ExecuteTemplate(w, "music_player", nil)

	// render music queue
	tmpl = template.Must(template.ParseFiles("templates/CurrentQueue.html"))
	tmpl.ExecuteTemplate(w, "room_queue", nil)
}

/*
	API
*/

// route: "GET /api/new-user"
func HandleNewUser(w http.ResponseWriter, r *http.Request) {
	userID := uuid.New()
	userIDByte := userID[:]
	encodedBase64 := base64.RawURLEncoding.EncodeToString(userIDByte)
	// slog.Debug("GET /new-user", "base64", encodedBase64)
	w.Write([]byte(encodedBase64))
}

// route: "POST /api/session"
func HandleNewSession(w http.ResponseWriter, r *http.Request) {
	room.ClientMapMutex.RLock()
	defer room.ClientMapMutex.RUnlock()
	// return a session id
	pUsername := r.PostFormValue("cfg_username")
	pUID := r.PostFormValue("user_id")

	// never trust the client
	decodedUID, err := base64.RawURLEncoding.DecodeString(pUID)
	uid, err := uuid.FromBytes(decodedUID)
	if err != nil {
		slog.Info("Invalid user UUID from client:", "status", http.StatusBadRequest, "err", err)
		http.Error(w, "", http.StatusBadRequest)
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
			slog.Info("Invalid room UUID from client:", "status", http.StatusBadRequest, "err", err)
			http.Error(w, "", http.StatusBadRequest)
			return
		}
	}

	// valid data
	// check user holds a sid already
	if _, ok := entryToken[uid]; ok {
		slog.Info("client has a token already", "endpoint", "POST /api/session", "uid", uid.String())
		http.Error(w, "", http.StatusForbidden)
		return
	}

	// check if client has a websocket connection already
	if c, ok := room.ClientMap[uid]; ok {
		slog.Info("client has already connected", "endpoint", "POST /api/session", "uid", c.ID.String())
		http.Error(w, "", http.StatusForbidden)
		return
	}

	// cache the user profile
	sid := uuid.New()
	profile := &UserProfile{
		name: name,
		uid:  uid,
		sid:  sid,
		rid:  rid,
	}
	entryProfiles[sid] = profile
	entryToken[uid] = &sid
	go profile.timeout()

	base64SID := base64.RawURLEncoding.EncodeToString(sid[:])
	w.Write([]byte(base64SID))
}

// route: "GET /api/create?sid="
func HandleCreateRoom(w http.ResponseWriter, r *http.Request) {
	sid, err := decodeQueryID(r, "sid")
	if err != nil {
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	// valid data
	userProfile, ok := entryProfiles[sid]
	if !ok {
		slog.Info("user profile not found", "status", http.StatusForbidden, "sid", sid.String())
		http.Error(w, "", http.StatusForbidden)
		return
	}
	if _, ok := room.NewHubs[sid]; ok {
		slog.Info("already created a new hub for client", "status", http.StatusTooManyRequests, "sid", sid.String())
		http.Error(w, "", http.StatusTooManyRequests)
		return
	}
	rid := uuid.New()
	userProfile.rid = rid

	hub := room.CreateHub(rid)
	room.HubMap[rid] = hub
	room.NewHubs[sid] = hub
	// reclaim memory when anything goes wrong
	go hub.Run()
	go hub.Timeout(&sid)

	base64RID := base64.RawURLEncoding.EncodeToString(rid[:])
	w.Write([]byte(base64RID))
}

// route: "GET /api/users?sid="
func UserList(w http.ResponseWriter, r *http.Request) {
	room.ClientMapMutex.RLock()
	room.TokenMapMutex.RLock()
	defer func() {
		room.TokenMapMutex.RUnlock()
		room.ClientMapMutex.RUnlock()
	}()

	sid, err := decodeQueryID(r, "sid")
	if err != nil {
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	var client *room.Client
	if uid, ok := room.TokenMap[sid]; ok {
		_client, ok := room.ClientMap[*uid]
		if !ok {
			slog.Info("client not found from sid", "sid", sid.String())
			http.Error(w, "", http.StatusInternalServerError)
			return
		}
		client = _client
	} else {
		slog.Info("token from client does not exists", "sid", sid.String())
		http.Error(w, "", http.StatusForbidden)
		return
	}

	userlist := make(map[string]string)
	for c := range client.Hub.Clients {
		userlist[c.ID.String()] = c.Name
	}

	json, err := json.Marshal(userlist)
	if err != nil {
		slog.Error("user list json encode error", "err", err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	w.Write(json)
}
