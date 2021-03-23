package broadcast

import "sync"

type Broadcaster struct {
	lock      sync.Mutex
	listeners []chan interface{}
}

func New() *Broadcaster {
	return &Broadcaster{}
}

func (b *Broadcaster) Listen() chan interface{} {
	c := make(chan interface{})
	b.listeners = append(b.listeners, c)
	return c
}

func (b *Broadcaster) Broadcast() {
	for _, c := range b.listeners {
		c <- ""
	}
}

func (b *Broadcaster) Close() {
	for _, c := range b.listeners {
		close(c)
	}
}
