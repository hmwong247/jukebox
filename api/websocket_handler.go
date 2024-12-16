package api

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}

// @TODO: sperate websocket hub
// route: /ws
func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WS upgrader: ", err)
		return
	}
	defer conn.Close()

	for {
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("WS read: ", err)
			return
		}

		log.Println("WS recv: ", msg)
		err = conn.WriteMessage(msgType, msg)
		if err != nil {
			log.Println("WS write: ", err)
			return
		}
	}
}
