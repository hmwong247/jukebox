package api

import (
	"encoding/base64"
	"log"
	"main/core/corewebsocket"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

// route: /ws
func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	queryParam := r.URL.Query()
	qSID := strings.TrimSpace(queryParam.Get("sid"))
	log.Printf("ws connect sid: %v\n", qSID)
	decodedSID, err := base64.RawURLEncoding.DecodeString(qSID)
	sid, err := uuid.FromBytes(decodedSID)
	if err != nil {
		log.Printf("ws: Trying to enter hub with invalid session UUID")
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	profile, ok := entryMap[sid]
	if !ok {
		log.Printf("ws: session does not exists")
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
	hub, ok := corewebsocket.HubMap[profile.rid]
	if !ok {
		log.Printf("ws: hub does not exists")
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	// switch to websocket
	conn, err := corewebsocket.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WS upgrader: ", err)
		return
	}

	// create client
	client := &corewebsocket.Client{
		Conn:       conn,
		Hub:        hub,
		ID:         profile.uid,
		Name:       profile.name,
		Permission: 7,
		Send:       make(chan []byte, 1024),
	}

	client.Hub.Register <- client
	go client.Read()
	go client.Write()
}
