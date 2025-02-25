package linkedlist

import (
	"errors"
	"fmt"
	"strings"
)

// WARNING: this package is not thread safe, users should take care of concurrency usage by themselves
// To iterate over a list (where l is a *List[T]):
// for n := l.Head(); n != nil; n = n.Next() {
// 		do something
// }

type Node[T comparable] struct {
	value      T
	prev, next *Node[T]
	l          *List[T]
}

func (n *Node[T]) Next() *Node[T] {
	if cur := n.next; n.l != nil && cur != &n.l.root {
		return cur
	}
	return nil
}

func (n *Node[T]) Prev() *Node[T] {
	if cur := n.prev; n.l != nil && cur != &n.l.root {
		return cur
	}
	return nil
}

func (n *Node[T]) Val() *T {
	return &n.value
}

func (n *Node[T]) Set(t *T) {
	n.value = *t
}

// doubly-linked list
// The implementation, same as the standard Go library,
// internally a list is a ring,
// such that root.prev is tail, and root.next is head
type List[T comparable] struct {
	root Node[T]
	size int
}

func (l *List[T]) Init() *List[T] {
	l.root.next = &l.root
	l.root.prev = &l.root
	l.size = 0
	return l
}

func New[T comparable]() *List[T] {
	return new(List[T]).Init()
}

func (l *List[T]) lazyInit() {
	if l.root.next == nil && l.size == 0 {
		l.Init()
	}
}

// insert a node after with type T
func (l *List[T]) insertAt(v *T, at *Node[T]) error {
	if at == nil || at.next == nil || at.prev == nil {
		return errors.New("invalid node provided")
	}
	l.lazyInit()
	newNode := &Node[T]{value: *v}
	newNode.prev = at
	newNode.next = at.next
	newNode.l = l

	at.next.prev = newNode
	at.next = newNode
	l.size++
	return nil
}

// func (l *List[T]) Append(values ...T) {
// 	for _, value := range values {
// 		newNode := &Node[T]{value: value}

// 		l.size++
// 	}
// }

func (l *List[T]) InsertHead(v *T) error {
	err := l.insertAt(v, &l.root)
	return err
}

func (l *List[T]) InsertTail(v *T) error {
	err := l.insertAt(v, l.root.prev)
	return err
}

func (l *List[T]) InsertAfter(v *T, node *Node[T]) error {
	err := l.insertAt(v, node)
	return err
}

func (l *List[T]) InsertBefore(v *T, node *Node[T]) error {
	err := l.insertAt(v, node.prev)
	return err
}

func (l *List[T]) Remove(node *Node[T]) error {
	if l.size == 0 {
		return errors.New("list is size of 0")
	}
	if node == nil || node.next == nil {
		return errors.New("null pointer given")
	}

	node.prev.next = node.next
	node.next.prev = node.prev
	node.prev = nil
	node.next = nil
	node.l = nil
	node = nil
	l.size--

	return nil
}

func (l *List[T]) Swap(n1, n2 *Node[T]) error {
	if n1 == nil || n2 == nil || n1.next == nil || n2.next == nil {
		str := fmt.Sprintf("null pointer error, n1: %v, n2: %v", n1, n2)
		return errors.New(str)
	}
	if n1 == n2 {
		return nil
	}
	if n1.l != n2.l {
		return errors.New("swaping nodes with different linked list")
	}

	if n1.next == n2 || n1.prev == n2 {
		if n1.prev == n2 {
			tmp := n1
			n1 = n2
			n2 = tmp
		}
		ptr := &Node[T]{}
		ptr.next = n2.next
		// ptr.prev = n2.prev

		n1.prev.next = n2
		n2.prev = n1.prev
		n2.next = n1

		n1.prev = n2
		n1.next = ptr.next
		ptr.next.prev = n1
	} else {
		ptr := &Node[T]{}
		ptr.next = n2.next
		ptr.prev = n2.prev

		n1.prev.next = n2
		n1.next.prev = n2
		n2.next = n1.next
		n2.prev = n1.prev

		ptr.prev.next = n1
		ptr.next.prev = n1
		n1.prev = ptr.prev
		n1.next = ptr.next
	}

	return nil
}

func (l *List[T]) Head() *Node[T] {
	if l.size == 0 {
		return nil
	}
	return l.root.next
}

func (l *List[T]) Tail() *Node[T] {
	if l.size == 0 {
		return nil
	}
	return l.root.prev
}

func (l *List[T]) Size() int {
	return l.size
}

func (l *List[T]) String() string {
	values := []string{}
	for n := l.Head(); n != nil; n = n.Next() {
		values = append(values, fmt.Sprintf("%v", n.value))
	}
	return strings.Join(values, ",")
}
