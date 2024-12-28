package main

import (
	"log/slog"
	"net/http"
	"os"

	"main/api"
)

func main() {
	// slog
	logOpt := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
	slogger := slog.New(slog.NewTextHandler(os.Stdout, logOpt))
	slog.SetDefault(slogger)
	slog.Info("start")

	mux := http.NewServeMux()

	// handle and serve static files
	jsFS := http.FileServer(http.Dir("scripts"))
	mux.Handle("GET /scripts/", http.StripPrefix("/scripts/", jsFS))
	nodeFS := http.FileServer(http.Dir("node_modules"))
	mux.Handle("GET /node_modules/", http.StripPrefix("/node_modules/", nodeFS))

	// handle room operations
	mux.HandleFunc("/", api.HandleRoot)
	mux.HandleFunc("GET /home", api.HandleDefault)
	mux.HandleFunc("GET /join", api.HandleJoin)
	mux.HandleFunc("GET /lobby", api.EnterLobby)
	mux.HandleFunc("GET /api/new-user", api.HandleNewUser)
	mux.HandleFunc("GET /api/create", api.HandleCreateRoom)
	mux.HandleFunc("POST /api/session", api.HandleNewSession)

	// WebSocket
	mux.HandleFunc("/ws", api.HandleWebSocket)

	slog.Error("server crashed", "err", http.ListenAndServe(":8080", mux))
}
