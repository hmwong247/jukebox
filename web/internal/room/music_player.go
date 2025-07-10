package room

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"main/internal/ytdlp"
	"main/utils/weaksync"
	"net/http"
	"sync"
)

// debug
func (mp *MusicPlayer) String() string {
	var curPlaying string
	var curID int
	if mp.CurNode != nil {
		curPlaying = mp.CurNode.InfoJson.FullTitle
		curID = mp.CurNode.ID
	} else {
		curPlaying = "nil"
		curID = -1
	}
	str := fmt.Sprintf("[MP STATUS] - playing: {ID: %v} {%v}", curID, curPlaying)

	return str
}

// methods are not safe by default
type MusicPlayer struct {
	hub         *Hub // maybe no need to keep reference
	Playlist    *Playlist
	fetchLock   *sync.Mutex
	NodeWGCnt   *weaksync.WaitGroupCnt
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
		NodeWGCnt:   weaksync.CreateWaitGroupCnt(),
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
			mp.fetchLock.Lock()
			mp.lazyInit(ctx)
			mp.fetchLock.Unlock()

		case <-mp.NextSong:
			mp.fetchLock.Lock()
			mp.CurNode = nil
			mp.AudioReader = nil
			if mp.Playlist.Size() > 0 {
				// the node could be preloading
				mp.next()
			}
			mp.NodeWGCnt.Done()
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

func (mp *MusicPlayer) lazyInit(mpctx context.Context) {
	if mp.CurNode == nil {
		// init state
		node := mp.Playlist.Head()
		if node == nil {
			slog.Error("[mp] playlist is empty")
			return
		}

		if len(node.AudioByte) == 0 {
			mp.download(mpctx, node)
			mp.next()
		}

		if mp.NodeWGCnt.Count() > 0 {
			mp.NodeWGCnt.Done()
		}
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

func (mp *MusicPlayer) MusicInfoList() []MusicInfo {
	mp.Playlist.RLock()
	defer mp.Playlist.RUnlock()

	ret := []MusicInfo{}
	if mp.CurNode != nil {
		ret = append(ret, *mp.CurNode)
	}
	for n := mp.Playlist.list.Head(); n != nil; n = n.Next() {
		fmt.Printf("n.val(): %v\n", **n.Val())
		ret = append(ret, **n.Val())
	}

	return ret
}
