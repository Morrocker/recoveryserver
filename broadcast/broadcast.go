package broadcast

import (
	"sync"

	"github.com/morrocker/recoveryserver/utils"
)

type Broadcaster struct {
	lock      sync.Mutex
	listeners map[string]*Listener
}

type Listener struct {
	id string
	b  *Broadcaster
	C  chan interface{}
}

func New() *Broadcaster {
	return &Broadcaster{
		listeners: make(map[string]*Listener),
	}
}

func (b *Broadcaster) Listen() *Listener {
	b.lock.Lock()
	defer b.lock.Unlock()
	newId := utils.RandString(8)
	l := &Listener{
		id: newId,
		C:  make(chan interface{}),
		b:  b,
	}
	b.listeners[newId] = l

	return l
}

func (b *Broadcaster) Close(id string) {
	b.lock.Lock()
	defer b.lock.Unlock()
	l, ok := b.listeners[id]
	if ok {
		close(l.C)
		delete(b.listeners, id)
	}
}

func (b *Broadcaster) Broadcast() {
	b.lock.Lock()
	defer b.lock.Unlock()
	for _, l := range b.listeners {
		l.C <- ""
	}
}

func (b *Broadcaster) CloseAll() {
	b.lock.Lock()
	defer b.lock.Unlock()
	for _, l := range b.listeners {
		close(l.C)
	}
}

func (l *Listener) Close() {
	l.b.Close(l.id)
}
