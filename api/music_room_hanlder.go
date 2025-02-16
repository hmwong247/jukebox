package api

import (
	"context"
	"log/slog"
	"main/core/room"
	"main/core/ytdlp"
	"net/http"
	"strings"
)

type wsInfoJson struct {
	ID int
	ytdlp.InfoJson
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

	// find user
	room.ClientMapMutex.RLock()
	room.TokenMapMutex.RLock()

	uid, ok := room.TokenMap[sid]
	if !ok {
		slog.Error("token not found", "sid", sid.String())
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
	client, ok := room.ClientMap[*uid]
	if !ok {
		slog.Error("client not found", "uid", uid.String())
		http.Error(w, "", http.StatusInternalServerError)
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
	status := ytdlp.JsonDownloader.Submit(ctx, &req)

	// http: mq response
	switch status {
	case http.StatusAccepted:
		w.WriteHeader(http.StatusAccepted)
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
			msg := room.Message{
				MsgType: 3,
				To:      client.ID,
				Data:    "timeout",
			}
			client.Hub.BroadcastMsg(msg)
			return
		case err := <-req.ErrCh:
			slog.Error("[api] info json err", "req", req, "err", err)
			msg := room.Message{
				MsgType: 3,
				To:      client.ID,
				Data:    "failed",
			}
			client.Hub.BroadcastMsg(msg)
			return
		case <-req.FinCh:
			// enqueue playlist
			node := room.MusicNode{
				URL:      pURL,
				InfoJson: req.Response,
			}
			if err := client.Hub.Playlist.Enqueue(&node); err != nil {
				slog.Error("[api] enqueue err", "err", err)
				return
			} else {
				client.Hub.Playlist.Traverse()
			}
			// broadcast json to websocket
			wsInfoJson := wsInfoJson{
				ID:       node.ID,
				InfoJson: req.Response,
			}
			msg := room.Message{
				MsgType:  3,
				UID:      client.ID.String(),
				Username: client.Name,
				Data:     wsInfoJson,
			}
			client.Hub.BroadcastMsg(msg)

			// notify hub
			client.Hub.AddedSong <- struct{}{}
		}
	}()
}
