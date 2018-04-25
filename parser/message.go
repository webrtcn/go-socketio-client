package parser

import (
	"io"
)

//MessageType message type
type MessageType int

//Const fields
const (
	MessageText MessageType = iota
	MessageBinary
)

type FrameReader interface {
	NextReader() (MessageType, io.ReadCloser, error)
}

type FrameWriter interface {
	NextWriter(MessageType) (io.WriteCloser, error)
}
