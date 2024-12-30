package api

import (
	"encoding/base64"
	"log/slog"
	"main/core/corewebsocket"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// route: /ws
func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	queryParam := r.URL.Query()
	qSID := strings.TrimSpace(queryParam.Get("sid"))
	decodedSID, err := base64.RawURLEncoding.DecodeString(qSID)
	sid, err := uuid.FromBytes(decodedSID)
	if err != nil {
		slog.Info("ws: Trying to enter hub with invalid session UUID", "qSID", qSID)
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	profile, ok := entryMap[sid]
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
	delete(entryMap, sid)
}
