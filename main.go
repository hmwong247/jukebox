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
	fs := http.FileServer(http.Dir("scripts"))
	mux.Handle("GET /scripts/", http.StripPrefix("/scripts/", fs))

	// handle room operations
	mux.HandleFunc("GET /", api.HandleRoot)
	mux.HandleFunc("GET /home", api.HandleDefault)
	mux.HandleFunc("GET /new-user", api.HandleNewUser)
	mux.HandleFunc("GET /join/{id}", api.HandleJoin)
	mux.HandleFunc("POST /join", api.HandleJoin)
	mux.HandleFunc("POST /create-room", api.HandleCreateRoom)

	log.Fatal(http.ListenAndServe(":8080", mux))
}
