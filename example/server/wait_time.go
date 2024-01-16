package server

import (
	"sort"
	"sync"
)

type WaitTime interface {
	Wait(deadline uint64) <-chan struct{}
	Trigger(deadline uint64)
}

var closec chan struct{}

func init() {
	closec = make(chan struct{})
	close(closec)
}

type item struct {
	key uint64
	ch  chan struct{}
}

type itemSlice []item

type timeList struct {
	sync.Mutex
	lastDeadline uint64
	items        itemSlice
}

func NewTimeList() WaitTime {
	return &timeList{
		items: make(itemSlice, 0, 128),
	}
}

func (t *timeList) Wait(deadline uint64) <-chan struct{} {
	t.Lock()
	defer t.Unlock()
	if t.lastDeadline >= deadline {
		return closec
	}

	i := sort.Search(len(t.items), func(i int) bool {
		return t.items[i].key >= deadline
	})
	if i < len(t.items) && t.items[i].key == deadline {
		return t.items[i].ch
	}
	it := item{
		key: deadline,
		ch:  make(chan struct{}),
	}
	if i == len(t.items) {
		t.items = append(t.items, it)
		return it.ch
	}
	t.items = append(t.items, it) // this for expand memory space
	copy(t.items[i+1:], t.items[i:len(t.items)-1])
	t.items[i] = it
	return it.ch
}

func (t *timeList) Trigger(deadline uint64) {
	t.Lock()
	defer t.Unlock()
	t.lastDeadline = deadline
	index := sort.Search(len(t.items), func(i int) bool {
		return t.items[i].key > deadline
	})

	for i := 0; i < index; i++ {
		close(t.items[i].ch)
	}
	if index == len(t.items) {
		t.items = t.items[0:0]
		return
	}
	copy(t.items[0:index], t.items[index:])
	t.items = t.items[0 : len(t.items)-index]
}
