package room

import (
	"context"
	"log/slog"
	"main/core/ytdlp"
	"net/http"
)

type MusicPlayer struct {
	hub                  *Hub // maybe no need to put this reference in the struct
	Playlist             *Playlist
	CurNode, PreloadNode *MusicInfo

	// playlist control channel
	AddedSong chan struct{}
	NextSong  chan struct{}
}

func CreateMusicPlayer() *MusicPlayer {
	playlist := NewPlaylist()

	return &MusicPlayer{
		Playlist:    playlist,
		CurNode:     nil,
		PreloadNode: nil,

		AddedSong: make(chan struct{}),
		NextSong:  make(chan struct{}),
	}
}

func (mp *MusicPlayer) Run(ctx context.Context, h *Hub) {
	defer func() {
		mp.Playlist.Clear()
		mp.hub = nil
	}()

	// set reference to hub
	mp.hub = h

	for {
		select {
		case <-ctx.Done():
			return

		case <-mp.AddedSong:
			// do something
			go mp.checkPlaylist(ctx)

		case <-mp.NextSong:
			// do something

		}
	}
}

func (mp *MusicPlayer) checkPlaylist(mpctx context.Context) {
	// if mp.CurNode == nil {
	ctx, cancel := context.WithTimeout(mpctx, ytdlp.TIMEOUT_AUDIO)
	defer cancel()
	node, err := mp.Playlist.Dequeue()
	if err != nil {
		slog.Error("[hub] dequeue playlist err", "err", err)
		return
	}
	req := ytdlp.RequestAudio{
		Ctx:   ctx,
		URL:   node.URL,
		ErrCh: make(chan error),
		FinCh: make(chan struct{}),
	}
	status, _ := ytdlp.AudioDownloader.Submit(ctx, &req)

	// mq response
	if status != http.StatusAccepted {
		slog.Info("[player] failed to enqueue request", "request", req)
		return
	}

	// audio byte response
	select {
	case <-ctx.Done():
		slog.Info("[player] request audio timeout")
		return
	case err := <-req.ErrCh:
		slog.Info("[player] audio byte reponse err", "req", req, "err", err)
		return
	case <-req.FinCh:
	}
	msg := DirectMessage[[]byte]{
		MsgType: MSG_EVENT_PLAYLIST,
		To:      mp.hub.Host.ID,
		Data:    req.Response,
	}
	mp.hub.DirectMsg(&msg)
	node.AudioByte = req.Response
	mp.CurNode = node
	// update node can send id,ok to host
	// }
}
