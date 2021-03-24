package recovery

func (r *Recovery) flowGate() bool {
	l := r.broadcaster.Listen()
	for {
		switch r.Status {
		case Running:
			l.Close()
			return false
		case Paused, Canceled:
			if r.Status == Canceled {
				l.Close()
				return true
			}
		}
		<-l.C
	}
}

func (r *Recovery) notify() {
	r.broadcaster.Broadcast()
}
