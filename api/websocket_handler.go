package api

import (
	"log/slog"
	"main/core/room"
	"net/http"
	"time"
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
		slog.Info("ws: Trying to enter hub with invalid session UUID")
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	// check if client has a connection already
	if uid, ok := room.TokenMap[sid]; ok {
		if c, ok := room.ClientMap[*uid]; ok {
			slog.Info("ws: client has already connected", "uid", c.ID.String())
			http.Error(w, "", http.StatusForbidden)
			return
		}
	}

	profile, ok := entryProfiles[sid]
	if !ok {
		slog.Error("ws: session does not exists", "status", http.StatusForbidden)
		http.Error(w, "", http.StatusForbidden)
		return
	}
	hub, ok := room.HubMap[profile.rid]
	if !ok {
		slog.Error("ws: hub does not exists", "status", http.StatusInternalServerError)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	// switch to websocket
	conn, err := room.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("ws upgrade err", "err", err)
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
	msg := room.Message{
		MsgType:  1,
		UID:      client.ID.String(),
		Username: client.Name,
		Data:     "join",
	}
	// broadcast before joining, avoid duplicating in client in frontend
	client.Hub.BroadcastMsg(msg)
	client.Hub.Register <- client
	go client.Read()
	go client.Write()

	// clean up the API entry cache
	delete(entryProfiles, sid)
	delete(entryToken, client.ID)
}
