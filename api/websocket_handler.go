package api

import (
	"log/slog"
	"main/core/corewebsocket"
	"net/http"
	"time"
)

// route: /ws
func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	sid, err := decodeQueryID(r, "sid")
	if err != nil {
		slog.Info("ws: Trying to enter hub with invalid session UUID")
		http.Error(w, "", http.StatusForbidden)
		return
	}

	// check if client has a connection already
	c, ok := corewebsocket.ClientMap[sid]
	if ok {
		slog.Info("ws: client has already connected", "cid", c.ID.String())
		http.Error(w, "", http.StatusForbidden)
		return
	}

	// run the websocket hub if it is a new hub
	newHub, ok := corewebsocket.NewHubs[sid]
	if ok {
		go newHub.Run()
		delete(corewebsocket.NewHubs, sid)
	}

	profile, ok := entryProfiles[sid]
	if !ok {
		slog.Error("ws: session does not exeists", "status", http.StatusForbidden)
		http.Error(w, "", http.StatusForbidden)
		return
	}
	hub, ok := corewebsocket.HubMap[profile.rid]
	if !ok {
		slog.Error("ws: hub does not exeists", "status", http.StatusInternalServerError)
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
		Name:          profile.name,
		Permission:    7,
		Send:          make(chan []byte, 1024),
		JoinUnixMilli: time.Now().UnixMilli(),
	}
	corewebsocket.ClientMap[sid] = client

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

	// clean up the entryMap
	delete(entryProfiles, sid)
}
