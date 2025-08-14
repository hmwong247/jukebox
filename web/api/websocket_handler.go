package api

import (
	"main/internal/room"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

// route: /ws?sid=
func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	room.ClientMapMutex.Lock()
	room.TokenMapMutex.Lock()
	defer func() {
		room.TokenMapMutex.Unlock()
		room.ClientMapMutex.Unlock()
	}()

	sid, err := decodeQueryID(r, "sid")
	if err != nil {
		log.Debug().
			Err(err).
			Msg("[ws] Failed to connect websocket")
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	// check if client has a connection already
	if uid, ok := room.TokenMap[sid]; ok {
		if c, ok := room.ClientMap[*uid]; ok {
			log.Debug().
				Str("uid", c.ID.String()).
				Str("rid", c.Hub.ID.String()).
				Msg("[ws] Client has already connected")
			http.Error(w, "", http.StatusForbidden)
			return
		}
	}

	profile, ok := entryProfiles[sid]
	if !ok {
		log.Debug().
			Str("sid", sid.String()).
			Msg("[ws] session does not exists")
		http.Error(w, "", http.StatusForbidden)
		return
	}
	hub, ok := room.HubMap[profile.rid]
	if !ok {
		log.Error().
			Any("profile", profile).
			Msg("[ws] hub does not exists")
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	// switch to websocket
	conn, err := room.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().
			Str("sid", sid.String()).
			Str("rid", profile.rid.String()).
			Str("uid", profile.uid.String()).
			Err(err).
			Msg("[ws] websocket upgrade error")
		return
	}

	// create client
	client := &room.Client{
		Conn:          conn,
		Hub:           hub,
		ID:            profile.uid,
		Token:         sid,
		Name:          profile.name,
		Permission:    7,
		Send:          make(chan []byte, 1024),
		JoinUnixMilli: time.Now().UnixMilli(),
	}
	room.ClientMap[client.ID] = client
	room.TokenMap[sid] = &client.ID

	// broadcast join notification
	msg := room.BroadcastMessage[room.Event]{
		MsgType:  room.MSG_EVENT_ROOM,
		UID:      client.ID.String(),
		Username: client.Name,
		Data:     "join",
	}
	// broadcast before joining, avoid duplicating in client in frontend
	client.Hub.BroadcastMsg(&msg)
	client.Hub.Register <- client
	go client.Read()
	go client.Write()

	// clean up the API entry cache
	delete(entryProfiles, sid)
	delete(entryToken, client.ID)
}
