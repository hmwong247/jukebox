package api

import (
	"log/slog"
	"main/core/corewebsocket"
	"net/http"
	"time"
)

// route: /ws?sid=
func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	corewebsocket.ClientMapMutex.Lock()
	corewebsocket.TokenMapMutex.Lock()
	defer func() {
		corewebsocket.TokenMapMutex.Unlock()
		corewebsocket.ClientMapMutex.Unlock()
	}()

	sid, err := decodeQueryID(r, "sid")
	if err != nil {
		slog.Info("ws: Trying to enter hub with invalid session UUID")
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	// check if client has a connection already
	if uid, ok := corewebsocket.TokenMap[sid]; ok {
		if c, ok := corewebsocket.ClientMap[*uid]; ok {
			slog.Info("ws: client has already connected", "uid", c.ID.String())
			http.Error(w, "", http.StatusForbidden)
			return
		}
	}

	// run the websocket hub if it is a new hub
	if newHub, ok := corewebsocket.NewHubs[sid]; ok {
		go newHub.Run()
		delete(corewebsocket.NewHubs, sid)
	}

	profile, ok := entryProfiles[sid]
	if !ok {
		slog.Error("ws: session does not exists", "status", http.StatusForbidden)
		http.Error(w, "", http.StatusForbidden)
		return
	}
	hub, ok := corewebsocket.HubMap[profile.rid]
	if !ok {
		slog.Error("ws: hub does not exists", "status", http.StatusInternalServerError)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	// switch to websocket
	conn, err := corewebsocket.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("ws upgrade err", "err", err)
		return
	}

	// create client
	client := &corewebsocket.Client{
		Conn:          conn,
		Hub:           hub,
		ID:            profile.uid,
		Token:         sid,
		Name:          profile.name,
		Permission:    7,
		Send:          make(chan []byte, 1024),
		JoinUnixMilli: time.Now().UnixMilli(),
	}
	corewebsocket.ClientMap[client.ID] = client
	corewebsocket.TokenMap[sid] = &client.ID

	// broadcast join notification
	msg := corewebsocket.Message{
		MsgType:  1,
		UID:      client.ID.String(),
		Username: client.Name,
		Data:     "join",
	}
	client.Hub.Broadcast <- msg
	client.Hub.Register <- client
	go client.Read()
	go client.Write()

	// clean up the API entry cache
	delete(entryProfiles, sid)
	delete(entryToken, client.ID)
}
