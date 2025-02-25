package room

import (
	"errors"
	"fmt"
	"main/core/ytdlp"
	"main/utils/linkedlist"
	"sync"
)

const (
	LIST_MAX_SIZE = 1024
)

type autoIncID struct {
	sync.Mutex
	id int
}

func (a *autoIncID) ID() int {
	a.Lock()
	defer a.Unlock()
	a.id++
	return a.id
}

type MusicInfo struct {
	ID        int
	URL       string
	AudioByte []byte
	InfoJson  ytdlp.InfoJson
}

// type MusicInfo is not comparable,
// pointer to pointer to MusicInfo is needed
type Playlist struct {
	sync.RWMutex
	list   *linkedlist.List[*MusicInfo]
	autoID autoIncID
}

func NewPlaylist() *Playlist {
	return &Playlist{
		list:   linkedlist.New[*MusicInfo](),
		autoID: autoIncID{id: -1},
	}
}

func (playlist *Playlist) Enqueue(info *MusicInfo) error {
	playlist.Lock()
	defer playlist.Unlock()

	if playlist.list.Size() >= LIST_MAX_SIZE {
		return errors.New("enqueue err: playlist reached max size")
	}

	info.ID = playlist.autoID.ID()
	if err := playlist.list.InsertTail(&info); err != nil {
		return err
	}

	return nil
}

func (playlist *Playlist) Remove(id int) error {
	playlist.Lock()
	defer playlist.Unlock()

	var n *linkedlist.Node[*MusicInfo] = nil
	for n = playlist.list.Head(); n != nil; n = n.Next() {
		if infoPtr := n.Val(); (*infoPtr).ID == id {
			break
		}
	}

	if n != nil {
		err := playlist.list.Remove(n)
		return err
	}

	return errors.New("id not found")
}

// return a copy of the music node
func (playlist *Playlist) Dequeue() (*MusicInfo, error) {
	playlist.Lock()
	defer playlist.Unlock()

	node := playlist.list.Head()
	info := node.Val()
	if err := playlist.list.Remove(node); err != nil {
		return nil, err
	}

	return *info, nil
}

func (playlist *Playlist) Clear() {
	playlist.Lock()
	defer playlist.Unlock()

	playlist.list.Init()
}

func (playlist *Playlist) Traverse() {
	playlist.RLock()
	defer playlist.RUnlock()

	fmt.Printf("playlist: %v\n", playlist.list)
}
