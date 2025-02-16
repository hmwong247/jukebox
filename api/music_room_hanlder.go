package api

import (
	"context"
	"log/slog"
	"main/core/room"
	"main/core/ytdlp"
	"net/http"
	"strings"
	"sync"
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
		Ctx:      ctx,
		URL:      pURL,
		Response: make(chan any),
	}
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cancel()

		// http: mq response
		select {
		case <-ctx.Done():
			w.WriteHeader(http.StatusRequestTimeout)
			slog.Debug("[api] mq response ctx done")
			return
		case res := <-req.Response:
			if status, ok := res.(int); ok {
				w.WriteHeader(status)
			} else {
				http.Error(w, "", http.StatusInternalServerError)
				slog.Debug("[api] mq response err, failed to cast int")
				return
			}
		}

		// websocket: json response
		select {
		case <-ctx.Done():
			slog.Debug("json response ctx done")
			msg := room.Message{
				MsgType: 3,
				To:      client.ID,
				Data:    "timeout",
			}
			client.Hub.BroadcastMsg(msg)
			return
		case res := <-req.Response:
			if json, ok := res.(ytdlp.InfoJson); ok {
				// enqueue playlist
				node := room.MusicNode{
					URL:      pURL,
					InfoJson: json,
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
					InfoJson: json,
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
			} else if err, ok := res.(error); ok {
				slog.Error("[api] info json err", "err", err)
				msg := room.Message{
					MsgType: 3,
					To:      client.ID,
					Data:    "failed",
				}
				client.Hub.BroadcastMsg(msg)
				return
			}
		}
	}()
	ytdlp.JsonDownloader.Submit(ctx, &req)
	wg.Wait()

	// w.Write([]byte(strconv.Itoa(node.ID)))
	// go fetchInfoJson(&node, client)
}
