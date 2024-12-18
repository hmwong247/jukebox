package api

import (
	"encoding/base64"
	"log"
	"main/core/corewebsocket"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var (
	upgrader = websocket.Upgrader{}
	HubMap   = make(map[uuid.UUID]*corewebsocket.Hub)
)

// route: /ws
func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	queryParam := r.URL.Query()
	qSID := strings.TrimSpace(queryParam.Get("sid"))
	log.Printf("ws connect: %v\n", qSID)
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
	hub, ok := HubMap[profile.rid]
	if !ok {
		hub = corewebsocket.NewHub(sid)
		HubMap[profile.rid] = hub
	}

	// switch to websocket
	conn, err := upgrader.Upgrade(w, r, nil)
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

	go hub.Run()
	go client.Read()
	go client.Write()

	// defer conn.Close()
	// for {
	// 	msgType, msg, err := conn.ReadMessage()
	// 	if err != nil {
	// 		log.Println("WS read: ", err)
	// 		return
	// 	}
	//
	// 	log.Println("WS recv: ", msg)
	// 	err = conn.WriteMessage(msgType, msg)
	// 	if err != nil {
	// 		log.Println("WS write: ", err)
	// 		return
	// 	}
	// }
}
