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
	sync.RWMutex // no need
	nodeList     *list.List
	nodeMap      map[int]*list.Element
	tail         int // auto increament
}

func CreatePlaylist() *Playlist {
	return &Playlist{
		nodeList: list.New(),
		nodeMap:  make(map[int]*list.Element),
		tail:     0,
	}
}

func (playlist *Playlist) Enqueue(node *MusicNode) error {
	playlist.Lock()
	defer playlist.Unlock()

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
	playlist.Lock()
	defer playlist.Unlock()

	elem, ok := playlist.nodeMap[id]
	if !ok {
		return errors.New("remove err: element not found")
	}
	playlist.nodeList.Remove(elem)
	delete(playlist.nodeMap, id)

	return nil
}

// return a copy of the music node
func (playlist *Playlist) Dequeue() (MusicNode, error) {
	playlist.Lock()
	defer playlist.Unlock()

	elem := playlist.nodeList.Front()
	node := elem.Value.(MusicNode)
	if _, ok := playlist.nodeMap[node.ID]; !ok {
		return MusicNode{}, errors.New("dequeue err: element not found")
	}

	playlist.nodeList.Remove(elem)
	delete(playlist.nodeMap, node.ID)

	return node, nil
}

// return a reference of the music node
func (playlist *Playlist) Head() (*MusicNode, error) {
	playlist.RLock()
	defer playlist.RUnlock()

	elem := playlist.nodeList.Front()
	node := elem.Value.(MusicNode)
	if _, ok := playlist.nodeMap[node.ID]; !ok {
		return &MusicNode{}, errors.New("dequeue err: element not found")
	}

	return &node, nil
}

func (playlist *Playlist) Clear() {
	playlist.Lock()
	defer playlist.Unlock()

	playlist.nodeList.Init()
	clear(playlist.nodeMap)
}

func (playlist *Playlist) Traverse() {
	playlist.RLock()
	defer playlist.RUnlock()

	for e := playlist.nodeList.Front(); e != nil; e = e.Next() {
		node := e.Value.(MusicNode)
		fmt.Println(node)
	}
}
