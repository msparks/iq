package notify

import "sync"

type Notifiee chan interface{}

type Notifier struct {
	mu sync.Mutex
	notifiees []Notifiee  // Guarded by mu.
}

func (n *Notifier) NewNotifiee() Notifiee {
	n.mu.Lock()
	defer n.mu.Unlock()

	c := make(Notifiee)
	n.notifiees = append(n.notifiees, c)
	return c
}

func (n *Notifier) CloseNotifiee(c Notifiee) {
	n.mu.Lock()
	defer n.mu.Unlock()

	var r []Notifiee
	for _, v := range n.notifiees {
		if v != c {
			r = append(r, v)
		} else {
			close(c)
		}
	}
	n.notifiees = r
}

func (n *Notifier) Notify(v interface{}) {
	n.mu.Lock()
	defer n.mu.Unlock()

	for _, notifiee := range n.notifiees {
		notifiee <-v
	}
}
