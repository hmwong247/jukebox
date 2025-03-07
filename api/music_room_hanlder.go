package api

import (
	"context"
	"log/slog"
	"main/core/mq"
	"main/core/room"
	"main/core/ytdlp"
	"net/http"
	"strconv"
	"strings"
)

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

	// find user
	room.ClientMapMutex.RLock()
	room.TokenMapMutex.RLock()

	uid, ok := room.TokenMap[sid]
	if !ok {
		slog.Error("token not found", "sid", sid.String())
		http.Error(w, "", http.StatusBadRequest)
		room.TokenMapMutex.RUnlock()
		room.ClientMapMutex.RUnlock()
		return
	}
	client, ok := room.ClientMap[*uid]
	if !ok {
		slog.Error("client not found", "uid", uid.String())
		http.Error(w, "", http.StatusBadRequest)
		room.TokenMapMutex.RUnlock()
		room.ClientMapMutex.RUnlock()
		return
	}

	room.TokenMapMutex.RUnlock()
	room.ClientMapMutex.RUnlock()

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
			client.Hub.Player.AddedSong <- struct{}{}
		}
	}()
}
