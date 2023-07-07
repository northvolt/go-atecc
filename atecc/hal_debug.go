package atecc

type halDebug struct {
	id   string
	l    Logger
	next HAL
}

func (h *halDebug) Read(p []byte) (int, error) {
	h.l.Printf("%5s >>  recv(%d)", h.id, cap(p))
	n, err := h.next.Read(p)
	h.l.Printf("%5s <<  recv %d(%d) %+v", h.id, n, len(p), err)
	if n > 0 {
		h.l.Printf("%s", hexDump(p[:n]))
	}
	return n, err
}

func (h *halDebug) Write(p []byte) (int, error) {
	h.l.Printf("%5s >>  send", h.id)
	if len(p) > 0 {
		h.l.Printf("%s", hexDump(p))
	}
	n, err := h.next.Write(p)
	h.l.Printf("%5s <<  send %d %+v", h.id, n, err)
	return n, err
}

func (h *halDebug) Idle() error {
	h.l.Printf("%5s >>  idle", h.id)
	err := h.next.Idle()
	h.l.Printf("%5s <<  idle %#v", h.id, err)
	return err
}

func (h *halDebug) Wake() error {
	h.l.Printf("%5s >>  wake", h.id)
	err := h.next.Wake()
	h.l.Printf("%5s <<  wake %#v", h.id, err)
	return err
}
