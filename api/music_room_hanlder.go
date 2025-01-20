package api

import (
	"log/slog"
	"main/core/corewebsocket"
	"main/core/playlist"
	"main/core/ytdlp"
	"net/http"
)

// route: "POST /api/enqueue"
func SubmittedtoQueue(w http.ResponseWriter, r *http.Request) {
	corewebsocket.ClientMapMutex.RLock()
	corewebsocket.TokenMapMutex.RLock()
	defer func() {
		corewebsocket.TokenMapMutex.RUnlock()
		corewebsocket.ClientMapMutex.RUnlock()
	}()

	sid, err := decodeQueryID(r, "sid")
	if err != nil {
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	pURL := r.PostFormValue("post_url")

	// find user
	uid, ok := corewebsocket.TokenMap[sid]
	if !ok {
		slog.Error("token not found", "sid", sid.String())
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
	client, ok := corewebsocket.ClientMap[*uid]
	if !ok {
		slog.Error("client not found", "uid", uid.String())
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	node := playlist.MusicNode{NodeURL: pURL}
	client.Hub.Playlist.Enqueue(node)

	client.Hub.Playlist.Traverse()

	// test
	audioByteArr, err := ytdlp.CmdStart(pURL)
	if err != nil {
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	w.Write(audioByteArr)
}
