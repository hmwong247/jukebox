package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"main/api"
	"main/internal/ytdlp"
	"main/utils/gzipped"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// zerolog
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.DateTime})
	logDebug := os.Getenv("LOG_LEVEL")
	if logDebug == "debug" {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
	log.Warn().Str("LOG_LEVEL", zerolog.GlobalLevel().String()).Msg("logger config: ")

	// maybe cancellable?
	dlpctx, dlpcancel := context.WithCancel(context.Background())
	defer dlpcancel()
	jsonctx := context.WithValue(dlpctx, "name", "json downloader")
	audioctx := context.WithValue(dlpctx, "name", "audio downloader")
	go ytdlp.JsonDownloader.Run(jsonctx)
	go ytdlp.AudioDownloader.Run(audioctx)

	mux := http.NewServeMux()

	appFS := gzipped.GzipFileServer(http.FileServer(http.Dir("app/dist")))
	mux.Handle("GET /assets/", appFS)

	// handle room operations
	mux.HandleFunc("/", api.HandleRoot)
	// views
	mux.HandleFunc("GET /home", api.HandleDefault)
	mux.HandleFunc("GET /join", api.HandleJoin)
	// api
	mux.HandleFunc("GET /api/new-user", api.HandleNewUser)
	mux.HandleFunc("POST /api/session", api.HandleNewSession)
	mux.HandleFunc("GET /api/create", api.HandleCreateRoom)
	mux.HandleFunc("GET /api/users", api.UserList)
	mux.HandleFunc("GET /api/playlist", api.Playlist)
	mux.HandleFunc("POST /api/enqueue", api.EnqueueURL)
	mux.HandleFunc("POST /api/queue", api.EditQueue)
	mux.HandleFunc("GET /api/stream", api.StreamAudio)
	mux.HandleFunc("GET /api/streamend", api.StreamEnd)
	mux.HandleFunc("GET /api/streampreload", api.StreamPreload)

	// WebSocket
	mux.HandleFunc("/ws", api.HandleWebSocket)

	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal().Err(err).Msg("Server panic")
	}
}
