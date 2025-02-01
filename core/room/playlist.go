package room

import (
	"container/list"
	"errors"
	"fmt"
	"main/core/ytdlp"
	"sync"
)

const (
	LIST_MAX_SIZE = 1024
)

type MusicNode struct {
	ID        int
	URL       string
	AudioByte []byte
	InfoJson  ytdlp.InfoJson
}

type Playlist struct {
	Lock     sync.RWMutex // no need
	nodeList *list.List
	nodeMap  map[int]*list.Element
	tail     int // auto increament

	// control channel
	// AddSong    chan *Request
	// RemoveSong chan *Request
	// Idle       chan bool
	// Destroy    chan int
}

func CreatePlaylist() *Playlist {
	return &Playlist{
		Lock:     sync.RWMutex{},
		nodeList: list.New(),
		nodeMap:  make(map[int]*list.Element),
		tail:     0,
		// AddSong:    make(chan *Request),
		// RemoveSong: make(chan *Request),
		// Idle:       make(chan bool),
		// Destroy:    make(chan int),
	}
}

// func (playlist *Playlist) Play() {
// 	defer func() {
// 		close(playlist.AddSong)
// 		close(playlist.RemoveSong)
// 		close(playlist.Idle)
// 	}()
// 	for {
// 		select {
// 		case r := <-playlist.AddSong:
// 		case r := <-playlist.RemoveSong:
// 		case cmd := <-playlist.Destroy:
// 			slog.Debug("playlist recieved c4", "cmd", cmd)
// 			close(playlist.Destroy)
// 			return
// 		}
// 	}
// }

func (playlist *Playlist) Enqueue(node *MusicNode) error {
	playlist.Lock.Lock()
	defer playlist.Lock.Unlock()

	if playlist.nodeList.Len() >= LIST_MAX_SIZE {
		return errors.New("enqueue err: playlist reached max size")
	}

	node.ID = playlist.tail
	elem := playlist.nodeList.PushBack(*node)
	playlist.nodeMap[playlist.tail] = elem
	playlist.tail++

	return nil
}

func (playlist *Playlist) Remove(id int) error {
	playlist.Lock.Lock()
	defer playlist.Lock.Unlock()

	elem, ok := playlist.nodeMap[id]
	if !ok {
		return errors.New("remove err: element not found")
	}
	playlist.nodeList.Remove(elem)
	delete(playlist.nodeMap, id)

	return nil
}

func (playlist *Playlist) Dequeue() error {
	playlist.Lock.Lock()
	defer playlist.Lock.Unlock()

	elem := playlist.nodeList.Front()
	node := elem.Value.(*MusicNode)
	if _, ok := playlist.nodeMap[node.ID]; !ok {
		return errors.New("dequeue err: element not found")
	}

	playlist.nodeList.Remove(elem)
	delete(playlist.nodeMap, node.ID)

	return nil
}

func (playlist *Playlist) Clear() {
	playlist.Lock.Lock()
	defer playlist.Lock.Unlock()

	playlist.nodeList.Init()
	clear(playlist.nodeMap)
}

func (playlist *Playlist) Traverse() {
	playlist.Lock.RLock()
	defer playlist.Lock.RUnlock()

	for e := playlist.nodeList.Front(); e != nil; e = e.Next() {
		node := e.Value.(MusicNode)
		fmt.Println(node)
	}
}
