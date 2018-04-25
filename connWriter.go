package client

import (
	"io"
	"sync"
)

type connWriter struct {
	io.WriteCloser
	locker *sync.Mutex
}

func newConnWriter(w io.WriteCloser, locker *sync.Mutex) *connWriter {
	return &connWriter{
		WriteCloser: w,
		locker:      locker,
	}
}

func (w *connWriter) Close() error {
	defer func() {
		if w.locker != nil {
			w.locker.Unlock()
			w.locker = nil
		}
	}()
	return w.WriteCloser.Close()
}
