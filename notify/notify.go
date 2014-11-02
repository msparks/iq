package notify

type Notifiee chan interface{}

type Notifier struct {
	notifiees []Notifiee
}

func (n *Notifier) NewNotifiee() Notifiee {
	c := make(Notifiee)
	n.notifiees = append(n.notifiees, c)
	return c
}

func (n *Notifier) CloseNotifiee(c Notifiee) {
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
	for _, notifiee := range n.notifiees {
		notifiee <-v
	}
}
