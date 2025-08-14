package room

import (
	"errors"
	"fmt"
	"main/internal/ytdlp"
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
	URL       string `json:"-"`
	AudioByte []byte `json:"-"`
	// InfoJson  ytdlp.InfoJson
	ytdlp.InfoJson
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

// linkedlist has no `move()` implementation, this is a workaround method
// this moves the target node to before the other node
func (playlist *Playlist) Move(id, other int) error {
	// search the node, ensure they exists
	playlist.RLock()
	defer playlist.RUnlock()
	var n1 *linkedlist.Node[*MusicInfo] = nil
	for n1 = playlist.list.Head(); n1 != nil; n1 = n1.Next() {
		if infoPtr := n1.Val(); (*infoPtr).ID == id {
			break
		}
	}
	if n1 == nil {
		return fmt.Errorf("node not found, id: %v", id)
	}
	var n2 *linkedlist.Node[*MusicInfo] = nil
	for n2 = playlist.list.Head(); n2 != nil; n2 = n2.Next() {
		if infoPtr := n2.Val(); (*infoPtr).ID == other {
			break
		}
	}
	if n2 == nil {
		return fmt.Errorf("node not found, other: %v", other)
	}

	// create a dummy node and swap them, then remove the dummy node
	info := &MusicInfo{}
	if err := playlist.list.InsertBefore(&info, n2); err != nil {
		return err
	}
	dummy := n2.Prev() // reference the node for later clean up
	if err := playlist.list.Swap(n1, dummy); err != nil {
		return err
	}
	if err := playlist.list.Remove(dummy); err != nil {
		return err
	}

	return nil
}

func (playlist *Playlist) Dequeue() (*MusicInfo, error) {
	playlist.Lock()
	defer playlist.Unlock()

	node := playlist.list.Head()
	if node == nil {
		err := fmt.Errorf("Trying to dequeue an empty playlist")
		return nil, err
	}

	info := node.Val()
	if err := playlist.list.Remove(node); err != nil {
		return nil, err
	}

	return *info, nil
}

func (playlist *Playlist) Head() *MusicInfo {
	playlist.RLock()
	defer playlist.RUnlock()

	node := playlist.list.Head()
	if node == nil {
		return nil
	}
	info := node.Val()

	return *info
}

func (playlist *Playlist) Clear() {
	playlist.Lock()
	defer playlist.Unlock()

	playlist.list.Init()
}

func (playlist *Playlist) Size() int {
	playlist.Lock()
	defer playlist.Unlock()

	return playlist.list.Size()
}

func (playlist *Playlist) Traverse() {
	playlist.RLock()
	defer playlist.RUnlock()

	fmt.Printf("playlist: %v\n", playlist.list)
}
