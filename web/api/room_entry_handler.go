package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"main/internal/room"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

const (
	TIMEOUT_API_ENTRY = 10 * time.Second
)

var (
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
		errf := fmt.Errorf("Invalid UUID: %v, err: %v", qID, err)
		return id, errf
	}

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
	http.ServeFile(w, r, "app/dist/index.html")
}

// route: "GET /join?rid="
func HandleJoin(w http.ResponseWriter, r *http.Request) {
	rid, err := decodeQueryID(r, "rid")
	if err != nil {
		http.Error(w, "", http.StatusBadRequest)
		return
	}
	if _, ok := room.HubMap[rid]; !ok {
		log.Info().Str("rid", rid.String()).Msg("Hub not found")
		http.Error(w, "", http.StatusForbidden)
		return
	}

	// tmpl := template.Must(template.ParseFiles("app/dist/index.html"))
	// tmpl.Execute(w, nil)
	http.ServeFile(w, r, "app/dist/index.html")
}

/*
	API
*/

// route: "GET /api/new-user"
func HandleNewUser(w http.ResponseWriter, r *http.Request) {
	userID := uuid.New().String()
	w.Write([]byte(userID))
}

// route: "POST /api/session"
func HandleNewSession(w http.ResponseWriter, r *http.Request) {
	room.ClientMapMutex.RLock()
	defer room.ClientMapMutex.RUnlock()
	// return a session id
	pUsername := r.PostFormValue("cfg_username")
	pUID := r.PostFormValue("user_id")

	// never trust the client
	uid, err := uuid.Parse(pUID)
	if err != nil {
		log.Debug().
			Err(err).
			Str("uid", pUID).
			Msg("Invalid user UUID from client")
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
			log.Debug().
				Err(err).
				Str("rid", _rid.String()).
				Msg("Invalid room UUID from client")
			http.Error(w, "", http.StatusBadRequest)
			return
		}
	}

	// valid data
	// check user holds a sid already
	if sid, ok := entryToken[uid]; ok {
		log.Debug().
			Str("uid", uid.String()).
			Str("sid", sid.String()).
			Msg("Client has a session token already")
		http.Error(w, "", http.StatusForbidden)
		return
	}

	// check if client has a websocket connection already
	if c, ok := room.ClientMap[uid]; ok {
		log.Warn().
			Str("uid", uid.String()).
			Str("rid", c.Hub.ID.String()).
			Msg("Client has a websocket connection already")
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
		log.Debug().
			Str("sid", sid.String()).
			Msg("Failed to create room, user profile not found")
		http.Error(w, "", http.StatusForbidden)
		return
	}
	if hub, ok := room.NewHubs[sid]; ok {
		log.Debug().
			Str("sid", sid.String()).
			Str("rid", hub.ID.String()).
			Msg("Created a new hub already, waiting for client")
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

type userJson struct {
	Name string `json:"name"`
	Host bool   `json:"host"`
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
			log.Debug().
				Str("sid", sid.String()).
				Str("uid", uid.String()).
				Msg("Hub found from sid but client not found from the hub")
			http.Error(w, "", http.StatusInternalServerError)
			return
		}
		client = _client
	} else {
		log.Debug().
			Str("sid", sid.String()).
			Msg("Token from client does not exists")
		http.Error(w, "", http.StatusForbidden)
		return
	}

	userlist := make(map[string]userJson)
	for c := range client.Hub.Clients {
		isHost := false
		if c.ID == c.Hub.Host.ID {
			isHost = true
		}
		userjson := userJson{
			Name: c.Name,
			Host: isHost,
		}
		userlist[c.ID.String()] = userjson
	}

	json, err := json.Marshal(userlist)
	if err != nil {
		log.Error().Err(err).Msg("Failed to encode userlist json")
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	w.Write(json)
}

// route: "GET /api/playlist?sid="
func Playlist(w http.ResponseWriter, r *http.Request) {
	sid, err := decodeQueryID(r, "sid")
	if err != nil {
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	room.TokenMapMutex.RLock()
	room.ClientMapMutex.RLock()
	defer func() {
		room.ClientMapMutex.RUnlock()
		room.TokenMapMutex.RUnlock()
	}()

	var client *room.Client
	if uid, ok := room.TokenMap[sid]; ok {
		_client, ok := room.ClientMap[*uid]
		if !ok {
			log.Debug().
				Str("sid", sid.String()).
				Str("uid", uid.String()).
				Msg("Hub found from sid but client not found from the hub")
			http.Error(w, "", http.StatusInternalServerError)
			return
		}
		client = _client
	} else {
		log.Debug().
			Str("sid", sid.String()).
			Msg("Token from client does not exists")
		http.Error(w, "", http.StatusForbidden)
		return
	}

	musicInfoList := client.Hub.Player.MusicInfoList()
	jsonList, err := json.Marshal(musicInfoList)
	if err != nil {
		log.Error().Err(err).Msg("Failed to encode playlist json")
		http.Error(w, "", http.StatusInternalServerError)
	}

	w.Write(jsonList)
}
