package api

import (
	"context"
	"log/slog"
	"main/core/room"
	"main/core/ytdlp"
	"net/http"
	"time"
)

// route: "POST /api/enqueue"
func EnqueueURL(w http.ResponseWriter, r *http.Request) {
	sid, err := decodeQueryID(r, "sid")
	if err != nil {
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	pURL := r.PostFormValue("post_url")

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

	// enqueue
	// node := room.MusicNode{URL: pURL}
	// enqueueRes := room.AddRequest{
	// 	URL:      pURL,
	// 	Client:   client,
	// 	Response: make(chan error),
	// }
	// client.Hub.AddSong <- &enqueueRes
	// err = <-enqueueRes.Response
	// if err != nil {
	// 	slog.Info("playlist enqueue error", "err", err)
	// 	http.Error(w, "", http.StatusTooManyRequests)
	// 	return
	// }
	// response 200 ok just to tell the client that the server has recieved
	// the request which is being processed, the result will be sent with websocket

	req := &ytdlp.RequestJson{
		URL:      pURL,
		Response: make(chan any),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	go func() {
		defer cancel()

		// mq response
		select {
		case <-ctx.Done():
			slog.Debug("mq response ctx done")
			return
		case res := <-req.Response:
			if status, ok := res.(int); ok {
				w.WriteHeader(status)
			} else {
				http.Error(w, "", http.StatusInternalServerError)
				return
			}
		}

		// json response
		select {
		case <-ctx.Done():
			slog.Debug("json response ctx done")
			return
		case res := <-req.Response:
			if json, ok := res.(ytdlp.InfoJson); ok {
				msg := room.Message{
					MsgType:  3,
					UID:      client.ID.String(),
					Username: client.Name,
					Data:     json,
				}
				go client.Hub.BroadcastMsg(msg)

				// enqueue playlist
				node := room.MusicNode{
					InfoJson: json,
				}
				if err := client.Hub.Playlist.Enqueue(&node); err != nil {
					slog.Error("enqueue err", "err", err)
					return
				} else {
					client.Hub.Playlist.Traverse()
				}
			} else if err, ok := res.(error); ok {
				slog.Error("info json err", "err", err)
				msg := room.Message{
					MsgType: 3,
					To:      client.ID,
					Data:    "failed",
				}
				go client.Hub.BroadcastMsg(msg)
				return
			} else {
				http.Error(w, "", http.StatusInternalServerError)
				return
			}
		}
	}()
	ytdlp.JsonDownloader.Submit(ctx, req)

	// w.Write([]byte(strconv.Itoa(node.ID)))
	// go fetchInfoJson(&node, client)
}
