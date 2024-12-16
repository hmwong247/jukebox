package main

import (
	"log"
	"net/http"

	"main/api"
)

func main() {
	log.Println("start")
	api.Debug_data()

	mux := http.NewServeMux()

	// handle and serve static files
	jsFS := http.FileServer(http.Dir("scripts"))
	mux.Handle("GET /scripts/", http.StripPrefix("/scripts/", jsFS))
	nodeFS := http.FileServer(http.Dir("node_modules"))
	mux.Handle("GET /node_modules/", http.StripPrefix("/node_modules/", nodeFS))

	// handle room operations
	mux.HandleFunc("/", api.HandleRoot)
	mux.HandleFunc("GET /home", api.HandleDefault)
	mux.HandleFunc("GET /api/new-user", api.HandleNewUser)
	mux.HandleFunc("GET /api/join/{id}", api.HandleJoin)
	mux.HandleFunc("POST /api/create", api.HandleCreateRoom)
	mux.HandleFunc("GET /lobby", api.EnterLobby)

	// WebSocket
	mux.HandleFunc("/ws", api.HandleWebSocket)

	log.Fatal(http.ListenAndServe(":8080", mux))
}
