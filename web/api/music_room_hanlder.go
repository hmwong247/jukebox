package api

import (
	"context"
	"encoding/json"
	"main/internal/room"
	"main/internal/taskq"
	"main/internal/ytdlp"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

func getClient(sid uuid.UUID) *room.Client {
	room.ClientMapMutex.RLock()
	room.TokenMapMutex.RLock()

	uid, ok := room.TokenMap[sid]
	if !ok {
		log.Debug().
			Str("sid", sid.String()).
			Msg("[api] Failed to find client, token not found")
		room.TokenMapMutex.RUnlock()
		room.ClientMapMutex.RUnlock()
		return nil
	}
	client, ok := room.ClientMap[*uid]
	if !ok {
		log.Warn().
			Str("uid", uid.String()).
			Msg("[api] Failed to find client, uid not found in the server, something went wrong?")
		room.TokenMapMutex.RUnlock()
		room.ClientMapMutex.RUnlock()
		return nil
	}

	room.TokenMapMutex.RUnlock()
	room.ClientMapMutex.RUnlock()

	return client
}

// route: "POST /api/enqueue?sid="
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
	req := ytdlp.RequestInfojson{
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
		log.Debug().Msg("[api] taskq response ctx timeout")
		cancel()
		return
	default:
		log.Error().Int("status", status).Msg("Uknown taskq Response")
		cancel()
		return
	}

	// websocket: json response
	// we would like to respond the client asap,
	// so the websocket response will be wrapped by a goroutine
	// and close the http reponse writer
	go func() {
		defer cancel()
		select {
		case <-ctx.Done():
			log.Debug().Msg("[api] json response ctx timeout")
			taskStatusJson := taskq.TaskStatus{
				Cmd:    taskq.STATUS_CMD_UPDATE,
				TaskID: taskID,
				Status: taskq.STATUS_STR_TIMEOUT,
			}
			msg := room.DirectMessage[taskq.TaskStatus]{
				MsgType: room.MSG_EVENT_PLAYLIST,
				To:      client.ID,
				Data:    taskStatusJson,
			}
			client.Hub.DirectMsg(&msg)
			return
		case err := <-req.ErrCh:
			log.Error().Err(err).
				Str("reqURL", req.URL).
				Msg("[api] InfoJson error")
			taskStatusJson := taskq.TaskStatus{
				Cmd:    taskq.STATUS_CMD_UPDATE,
				TaskID: taskID,
				Status: taskq.STATUS_STR_FAILED,
			}
			msg := room.DirectMessage[taskq.TaskStatus]{
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
				log.Error().Err(err).Msg("[api] Enqueue URL error")
				return
			}

			// responds ok to client
			taskStatusJson := taskq.TaskStatus{
				Cmd:    taskq.STATUS_CMD_UPDATE,
				TaskID: taskID,
				Status: taskq.STATUS_STR_OK,
			}
			dmsg := room.DirectMessage[taskq.TaskStatus]{
				MsgType: room.MSG_EVENT_PLAYLIST,
				To:      client.ID,
				Data:    taskStatusJson,
			}
			client.Hub.DirectMsg(&dmsg)

			// broadcast json to websocket
			wsInfoJson := room.WSInfoJson{
				ID:       node.ID,
				Cmd:      room.INFOJSON_CMD_ADD,
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
	log.Debug().Str("MP status", client.Hub.Player.String()).Str("rid", client.Hub.B64ID()).Msg("[api] stream")

	// byte serve the audio
	client.Hub.Player.NodeWGCnt.Wait()
	reader := client.Hub.Player.AudioReader
	if reader == nil {
		log.Warn().
			Str("rid", client.Hub.B64ID()).
			Str("client id", client.B64ID()).
			Msg("MP byte reader is nil")
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
	log.Debug().Str("MP status", client.Hub.Player.String()).Str("rid", client.Hub.B64ID()).Msg("[api] streampreload")

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
	log.Debug().Str("MP status", client.Hub.Player.String()).Str("rid", client.Hub.B64ID()).Msg("[api] streamend")

	// if client is host?
	if client.Hub.Host.ID != client.ID {
		http.Error(w, "", http.StatusForbidden)
		return
	}
	client.Hub.Player.NodeWGCnt.Add(1)
	client.SignalMPNext()
}

type QueueAction struct {
	Cmd    string
	NodeID int
}

// route: "POST /api/queue?sid="
func EditQueue(w http.ResponseWriter, r *http.Request) {
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

	var queueAction QueueAction
	err = json.NewDecoder(r.Body).Decode(&queueAction)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// slog.Debug("[api] /api/queue", "queueAction", queueAction)

	err = client.Hub.Player.Playlist.Remove(queueAction.NodeID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// websocket: json response
	wsInfoJson := room.WSInfoJson{
		ID:  queueAction.NodeID,
		Cmd: room.INFOJSON_CMD_REMOVE,
	}
	msg := room.BroadcastMessage[room.WSInfoJson]{
		MsgType:  room.MSG_EVENT_PLAYLIST,
		UID:      client.ID.String(),
		Username: client.Name,
		Data:     wsInfoJson,
	}
	client.Hub.BroadcastMsg(&msg)

}
