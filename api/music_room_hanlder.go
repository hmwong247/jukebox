package api

import (
	"log/slog"
	"main/core/room"
	"main/core/ytdlp"
	"net/http"
	"strconv"
)

// route: "POST /api/enqueue"
func EnqueueURL(w http.ResponseWriter, r *http.Request) {
	// corewebsocket.ClientMapMutex.RLock()
	// corewebsocket.TokenMapMutex.RLock()
	// defer func() {
	// 	corewebsocket.TokenMapMutex.RUnlock()
	// 	corewebsocket.ClientMapMutex.RUnlock()
	// }()

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
	node := room.MusicNode{URL: pURL}
	enqueueRes := room.Request{
		Node:     &node,
		Client:   client,
		Response: make(chan error),
	}
	client.Hub.AddSong <- &enqueueRes
	err = <-enqueueRes.Response
	if err != nil {
		slog.Info("playlist enqueue error", "err", err)
		http.Error(w, "", http.StatusTooManyRequests)
		return
	}
	// response 200 ok just to tell the client that the server has recieved
	// the request which is being processed, the result will be sent with websocket
	w.Write([]byte(strconv.Itoa(node.ID)))

	go fetchInfoJson(&node, client)
}

func fetchInfoJson(node *room.MusicNode, client *room.Client) {
	// audioByteArr, err := ytdlp.CmdStart(pURL)
	// if err != nil {
	// 	http.Error(w, "", http.StatusBadRequest)
	// 	return
	// }
	// byteReader := bytes.NewReader(audioByteArr)
	// w.Write(audioByteArr)
	// http.ServeContent(w, r, "", time.Time{}, byteReader)

	infoJson, err := ytdlp.DownloadInfoJson(node.URL)
	if err != nil {
		slog.Error("info json err", "err", err)
		msg := room.Message{
			MsgType: 3,
			To:      client.ID,
			Data:    "failed",
		}
		go client.Hub.BroadcastMsg(msg)
		return
	}
	infoJson.ID = node.ID
	node.InfoJson = infoJson

	msg := room.Message{
		MsgType:  3,
		UID:      client.ID.String(),
		Username: client.Name,
		Data:     infoJson,
	}
	go client.Hub.BroadcastMsg(msg)
}
