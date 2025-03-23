package room

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"main/internal/ytdlp"
	"net/http"
	"sync"
)

// debug
func (mp *MusicPlayer) String() string {
	var curplaying string
	if mp.CurNode != nil {
		curplaying = mp.CurNode.InfoJson.FullTitle
	} else {
		curplaying = "nil"
	}
	var prefetched bool = false
	if mp.Playlist.Size() > 0 && len(mp.Playlist.Head().AudioByte) > 0 {
		prefetched = true
	}
	str := fmt.Sprintf("MP STATUS - currently playing: %v, prefetched: %v", curplaying, prefetched)

	return str
}

type MusicPlayer struct {
	hub         *Hub // maybe no need to keep reference
	Playlist    *Playlist
	fetchLock   *sync.Mutex
	CurNode     *MusicInfo
	AudioReader *bytes.Reader

	// playlist control channel
	AddedSong chan struct{}
	NextSong  chan struct{}
	Preload   chan struct{}
}

// json response to notify host
type MPStatus struct {
	NextID int
	OK     bool
}

func CreateMusicPlayer() *MusicPlayer {
	playlist := NewPlaylist()

	return &MusicPlayer{
		Playlist:    playlist,
		fetchLock:   &sync.Mutex{},
		CurNode:     nil,
		AudioReader: nil,

		AddedSong: make(chan struct{}),
		NextSong:  make(chan struct{}),
		Preload:   make(chan struct{}),
	}
}

func (mp *MusicPlayer) NewAudioReader() *bytes.Reader {
	var reader *bytes.Reader
	if mp.CurNode != nil {
		reader = bytes.NewReader(mp.CurNode.AudioByte)
	}

	return reader
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
			// slog.Debug("[mp] added", "status", mp)
			mp.checkState(ctx)

		case <-mp.NextSong:
			mp.fetchLock.Lock()
			mp.CurNode = nil
			mp.AudioReader = nil
			if mp.Playlist.Size() > 0 {
				// the node could be preloading
				mp.next()
			}
			mp.fetchLock.Unlock()
			// slog.Debug("[mp] next", "status", mp)

		case <-mp.Preload:
			// preloading the next song should not block the player
			go func() {
				if mp.Playlist.Size() > 0 {
					mp.fetchLock.Lock()
					node := mp.Playlist.Head()
					mp.download(ctx, node)
					mp.fetchLock.Unlock()
				}
				// slog.Debug("[mp] preload", "status", mp)
			}()
		}
	}
}

func (mp *MusicPlayer) checkState(mpctx context.Context) {
	if mp.CurNode == nil {
		// init state
		node := mp.Playlist.Head()
		if node == nil {
			slog.Error("[mp] playlist is empty")
			return
		}

		mp.fetchLock.Lock()
		if len(node.AudioByte) == 0 {
			mp.download(mpctx, node)
			mp.next()
		}
		mp.fetchLock.Unlock()
	}
}

func (mp *MusicPlayer) download(mpctx context.Context, node *MusicInfo) {
	if node == nil {
		slog.Error("[mp] trying to download audio to a nil target")
	} else {
		ctx, cancel := context.WithTimeout(mpctx, ytdlp.TIMEOUT_AUDIO)
		defer cancel()
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

		// update node can send id,ok to host
		node.AudioByte = req.Response

		mpstatus := MPStatus{
			NextID: node.ID,
			OK:     true,
		}
		msg := DirectMessage[MPStatus]{
			MsgType: MSG_EVENT_PLAYER,
			To:      mp.hub.Host.ID,
			Data:    mpstatus,
		}
		mp.hub.DirectMsg(&msg)
	}
}

func (mp *MusicPlayer) next() {
	// ensure all client finished the audio

	nextNode, err := mp.Playlist.Dequeue()
	if err != nil {
		slog.Error("[mp] dequeue error in next()", "err", err)
		return
	}
	mp.CurNode = nextNode
	mp.AudioReader = mp.NewAudioReader()
}
