package client

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/webrtcn/go-socketio-client/parser"
)

type encoder struct {
	w   parser.FrameWriter
	err error
}

func newEncoder(w parser.FrameWriter) *encoder {
	return &encoder{
		w: w,
	}
}

func (e *encoder) Encode(v packet) error {
	attachments := encodeAttachments(v.Data)
	v.attachNumber = len(attachments)
	if v.attachNumber > 0 {
		v.Type += _BINARY_EVENT - _EVENT
	}
	if err := e.encodePacket(v); err != nil {
		return err
	}
	for _, a := range attachments {
		if err := e.writeBinary(a); err != nil {
			return err
		}
	}
	return nil
}

func (e *encoder) encodePacket(v packet) error {
	writer, err := e.w.NextWriter(parser.MessageText)
	if err != nil {
		return err
	}
	defer writer.Close()
	w := newTrimWriter(writer, "\n")
	wh := newWriterHelper(w)
	wh.Write([]byte{byte(v.Type) + '0'})
	if v.Type == _BINARY_EVENT || v.Type == _BINARY_ACK {
		wh.Write([]byte(fmt.Sprintf("%d-", v.attachNumber)))
	}
	needEnd := false
	if v.NSP != "" {
		wh.Write([]byte(v.NSP))
		needEnd = true
	}
	if v.Id >= 0 {
		f := "%d"
		if needEnd {
			f = ",%d"
			needEnd = false
		}
		wh.Write([]byte(fmt.Sprintf(f, v.Id)))
	}
	if v.Data != nil {
		if needEnd {
			wh.Write([]byte{','})
			needEnd = false
		}
		if wh.Error() != nil {
			return wh.Error()
		}
		encoder := json.NewEncoder(w)
		return encoder.Encode(v.Data)
	}
	return wh.Error()
}

func (e *encoder) writeBinary(r io.Reader) error {
	writer, err := e.w.NextWriter(parser.MessageBinary)
	if err != nil {
		return err
	}
	defer writer.Close()

	if _, err := io.Copy(writer, r); err != nil {
		return err
	}
	return nil
}
