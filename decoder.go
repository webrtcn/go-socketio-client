package client

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"strconv"

	"github.com/webrtcn/go-socketio-client/parser"
)

type decoder struct {
	reader        parser.FrameReader
	message       string
	current       io.Reader
	currentCloser io.Closer
}

func newDecoder(r parser.FrameReader) *decoder {
	return &decoder{
		reader: r,
	}
}

func (d *decoder) Close() {
	if d != nil && d.currentCloser != nil {
		d.currentCloser.Close()
		d.current = nil
		d.currentCloser = nil
	}
}

//Decode decode the message except '_CONNECT','_DISCONNECT' message
func (d *decoder) Decode(v *packet) error {
	ty, r, err := d.reader.NextReader()
	if err != nil {
		return err
	}
	if d.current != nil {
		d.Close()
	}
	defer func() {
		if d.current == nil {
			r.Close()
		}
	}()

	if ty != parser.MessageText {
		return fmt.Errorf("need text package")
	}
	reader := bufio.NewReader(r)
	v.Id = -1
	t, err := reader.ReadByte()
	if err != nil {
		return err
	}
	v.Type = packetType(t - '0') //'0' (char) = 48 (byte)
	if v.Type == _BINARY_EVENT || v.Type == _BINARY_ACK {
		num, err := reader.ReadBytes('-')
		if err != nil {
			return err
		}
		numLen := len(num)
		if numLen == 0 {
			return fmt.Errorf("invalid packet")
		}
		n, err := strconv.ParseInt(string(num[:numLen-1]), 10, 64)
		if err != nil {
			return fmt.Errorf("invalid packet")
		}
		v.attachNumber = int(n)
	}

	next, err := reader.Peek(1)
	if err == io.EOF {
		return nil
	}
	if err != nil {
		return err
	}
	if len(next) == 0 {
		return fmt.Errorf("invalid packet")
	}

	if next[0] == '/' {
		path, err := reader.ReadBytes(',')
		if err != nil && err != io.EOF {
			return err
		}
		pathLen := len(path)
		if pathLen == 0 {
			return fmt.Errorf("invalid packet")
		}
		if err == nil {
			path = path[:pathLen-1]
		}
		v.NSP = string(path)
		if err == io.EOF {
			return nil
		}
	}

	id := bytes.NewBuffer(nil)
	finish := false
	for {
		next, err := reader.Peek(1)
		if err == io.EOF {
			finish = true
			break
		}
		if err != nil {
			return err
		}
		if '0' <= next[0] && next[0] <= '9' {
			if err := id.WriteByte(next[0]); err != nil {
				return err
			}
		} else {
			break
		}
		reader.ReadByte()
	}
	if id.Len() > 0 {
		id, err := strconv.ParseInt(id.String(), 10, 64)
		if err != nil {
			return err
		}
		v.Id = int(id)
	}
	if finish {
		return nil
	}
	switch v.Type {
	case _EVENT:
		fallthrough
	case _BINARY_EVENT, _CONNECT:
		msgReader, err := newMessageReader(reader)
		if err != nil {
			return err
		}
		d.message = msgReader.Message()
		d.current = msgReader
		d.currentCloser = r
	case _ACK:
		fallthrough
	case _BINARY_ACK:
		d.current = reader
		d.currentCloser = r
	}
	return nil
}

func (d *decoder) Message() string {
	return d.message
}

func (d *decoder) DecodeData(v *packet) error {
	if d.current == nil {
		return nil
	}
	defer func() {
		d.Close()
	}()
	decoder := json.NewDecoder(d.current)
	if err := decoder.Decode(v.Data); err != nil {
		return err
	}
	if v.Type == _BINARY_EVENT || v.Type == _BINARY_ACK {
		binary, err := d.decodeBinary(v.attachNumber)
		if err != nil {
			return err
		}
		if err := decodeAttachments(v.Data, binary); err != nil {
			return err
		}
		v.Type -= _BINARY_EVENT - _EVENT
	}
	return nil
}

func (d *decoder) decodeBinary(num int) ([][]byte, error) {
	ret := make([][]byte, num)
	for i := 0; i < num; i++ {
		d.currentCloser.Close()
		t, r, err := d.reader.NextReader()
		if err != nil {
			return nil, err
		}
		d.currentCloser = r
		if t == parser.MessageText {
			return nil, fmt.Errorf("need binary")
		}
		b, err := ioutil.ReadAll(r)
		if err != nil {
			return nil, err
		}
		ret[i] = b
	}
	return ret, nil
}
