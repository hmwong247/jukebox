package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"main/api"
	"main/core/ytdlp"
)

func main() {
	// slog
	logOpt := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
	slogger := slog.New(slog.NewTextHandler(os.Stdout, logOpt))
	slog.SetDefault(slogger)
	slog.Info("start")

	// maybe cancellable?
	dlpctx, dlpcancel := context.WithCancel(context.Background())
	defer dlpcancel()
	jsonctx := context.WithValue(dlpctx, "name", "json downloader")
	audioctx := context.WithValue(dlpctx, "name", "audio downloader")
	go ytdlp.JsonDownloader.Run(jsonctx)
	go ytdlp.AudioDownloader.Run(audioctx)

	mux := http.NewServeMux()

	// handle and serve static files
	jsFS := http.FileServer(http.Dir("scripts"))
	mux.Handle("GET /scripts/", http.StripPrefix("/scripts/", jsFS))
	nodeFS := http.FileServer(http.Dir("node_modules"))
	mux.Handle("GET /node_modules/", http.StripPrefix("/node_modules/", nodeFS))
	styleFS := http.FileServer(http.Dir("styles"))
	mux.Handle("GET /styles/", http.StripPrefix("/styles/", styleFS))

	// handle room operations
	mux.HandleFunc("/", api.HandleRoot)
	// views
	mux.HandleFunc("GET /home", api.HandleDefault)
	mux.HandleFunc("GET /join", api.HandleJoin)
	mux.HandleFunc("GET /lobby", api.EnterLobby)
	// api
	mux.HandleFunc("GET /api/new-user", api.HandleNewUser)
	mux.HandleFunc("POST /api/session", api.HandleNewSession)
	mux.HandleFunc("GET /api/create", api.HandleCreateRoom)
	mux.HandleFunc("GET /api/users", api.UserList)
	mux.HandleFunc("POST /api/enqueue", api.EnqueueURL)
	mux.HandleFunc("GET /api/stream", api.StreamAudio)
	mux.HandleFunc("GET /api/streamend", api.StreamEnd)
	mux.HandleFunc("GET /api/streampreload", api.StreamPreload)

	// WebSocket
	mux.HandleFunc("/ws", api.HandleWebSocket)

	slog.Error("server crashed", "err", http.ListenAndServe(":8080", mux))
}
