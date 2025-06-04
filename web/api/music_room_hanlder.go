package api

import (
	"context"
	"log/slog"
	"main/internal/mq"
	"main/internal/room"
	"main/internal/ytdlp"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

func getClient(sid uuid.UUID) *room.Client {
	room.ClientMapMutex.RLock()
	room.TokenMapMutex.RLock()

	uid, ok := room.TokenMap[sid]
	if !ok {
		slog.Error("token not found", "sid", sid.String())
		room.TokenMapMutex.RUnlock()
		room.ClientMapMutex.RUnlock()
		return nil
	}
	client, ok := room.ClientMap[*uid]
	if !ok {
		slog.Error("client not found", "uid", uid.String())
		room.TokenMapMutex.RUnlock()
		room.ClientMapMutex.RUnlock()
		return nil
	}

	room.TokenMapMutex.RUnlock()
	room.ClientMapMutex.RUnlock()

	return client
}

// route: "POST /api/enqueue"
func EnqueueURL(w http.ResponseWriter, r *http.Request) {
	sid, err := decodeQueryID(r, "sid")
	if err != nil {
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	pURL := strings.TrimSpace(r.PostFormValue("post_url"))
	if len(pURL) == 0 {
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	client := getClient(sid)
	if client == nil {
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	// respond 202 just to tell the client that the server has recieved
	// the request which is being processed, the result will be sent with websocket
	ctx, cancel := context.WithTimeout(context.Background(), ytdlp.TIMEOUT_JSON)
	req := ytdlp.RequestJson{
		Ctx:   ctx,
		URL:   pURL,
		ErrCh: make(chan error),
		FinCh: make(chan struct{}),
	}
	status, taskID := ytdlp.JsonDownloader.Submit(ctx, &req)

	// http: mq response
	switch status {
	case http.StatusAccepted:
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(strconv.FormatInt(taskID, 10)))
	case http.StatusTooManyRequests:
		w.WriteHeader(http.StatusTooManyRequests)
		cancel()
		return
	case http.StatusRequestTimeout:
		w.WriteHeader(http.StatusRequestTimeout)
		slog.Debug("[api] mq response ctx timeout")
		cancel()
		return
	default:
		slog.Error("UNKNOWN MQ Response", "status", status)
		cancel()
		return
	}

	// websocket: json response
	// we would like to repond the client asap,
	// so the websocket response will be wrapped by a goroutine
	// and close the http reponse writer
	go func() {
		defer cancel()
		select {
		case <-ctx.Done():
			slog.Debug("[api] json response ctx timeout")
			taskStatusJson := mq.TaskStatus{
				TaskID: taskID,
				Status: "timeout",
			}
			msg := room.DirectMessage[mq.TaskStatus]{
				MsgType: room.MSG_EVENT_PLAYLIST,
				To:      client.ID,
				Data:    taskStatusJson,
			}
			client.Hub.DirectMsg(&msg)
			return
		case err := <-req.ErrCh:
			slog.Error("[api] info json err", "req", req, "err", err)
			taskStatusJson := mq.TaskStatus{
				TaskID: taskID,
				Status: "failed",
			}
			msg := room.DirectMessage[mq.TaskStatus]{
				MsgType: room.MSG_EVENT_PLAYLIST,
				To:      client.ID,
				Data:    taskStatusJson,
			}
			client.Hub.DirectMsg(&msg)
			return
		case <-req.FinCh:
			// enqueue playlist
			node := room.MusicInfo{
				URL:      pURL,
				InfoJson: req.Response,
			}
			if err := client.Hub.Player.Playlist.Enqueue(&node); err != nil {
				slog.Error("[api] enqueue err", "err", err)
				return
			} else {
				client.Hub.Player.Playlist.Traverse()
			}

			// responds ok to client
			taskStatusJson := mq.TaskStatus{
				TaskID: taskID,
				Status: "ok",
			}
			dmsg := room.DirectMessage[mq.TaskStatus]{
				MsgType: room.MSG_EVENT_PLAYLIST,
				To:      client.ID,
				Data:    taskStatusJson,
			}
			client.Hub.DirectMsg(&dmsg)

			// broadcast json to websocket
			wsInfoJson := room.WSInfoJson{
				ID:       node.ID,
				Cmd:      "add",
				InfoJson: req.Response,
			}
			msg := room.BroadcastMessage[room.WSInfoJson]{
				MsgType:  room.MSG_EVENT_PLAYLIST,
				UID:      client.ID.String(),
				Username: client.Name,
				Data:     wsInfoJson,
			}
			client.Hub.BroadcastMsg(&msg)

			// notify hub
			// client.Hub.Player.AddedSong <- struct{}{}
			client.SignalMPAdd()
		}
	}()
}

// route: "GET /api/stream?sid="
func StreamAudio(w http.ResponseWriter, r *http.Request) {
	sid, err := decodeQueryID(r, "sid")
	if err != nil {
		http.Error(w, "", http.StatusBadRequest)
		return
	}
	client := getClient(sid)
	if client == nil {
		http.Error(w, "", http.StatusBadRequest)
		return
	}
	slog.Debug("[api] stream", "mp status", client.Hub.Player)

	// byte serve the audio
	reader := client.Hub.Player.AudioReader
	if reader == nil {
		slog.Warn("byte reader is nil", "hub id", client.Hub.B64ID(), "client id", client.B64ID())
		http.Error(w, "", http.StatusNotFound)
		return
	}

	http.ServeContent(w, r, "", time.Time{}, reader)
}

// route: "GET /api/streampreload?sid="
func StreamPreload(w http.ResponseWriter, r *http.Request) {
	sid, err := decodeQueryID(r, "sid")
	if err != nil {
		http.Error(w, "", http.StatusBadRequest)
		return
	}
	client := getClient(sid)
	if client == nil {
		http.Error(w, "", http.StatusBadRequest)
		return
	}
	slog.Debug("[api] streampreload", "mp status", client.Hub.Player)

	// if client is host?
	if client.Hub.Host.ID != client.ID {
		http.Error(w, "", http.StatusForbidden)
		return
	}
	client.SignalMPPreload()
}

// route: "GET /api/streamend?sid="
func StreamEnd(w http.ResponseWriter, r *http.Request) {
	sid, err := decodeQueryID(r, "sid")
	if err != nil {
		http.Error(w, "", http.StatusBadRequest)
		return
	}
	client := getClient(sid)
	if client == nil {
		http.Error(w, "", http.StatusBadRequest)
		return
	}
	slog.Debug("[api] streamend", "mp status", client.Hub.Player)

	// if client is host?
	if client.Hub.Host.ID != client.ID {
		http.Error(w, "", http.StatusForbidden)
		return
	}
	client.SignalMPNext()
}
