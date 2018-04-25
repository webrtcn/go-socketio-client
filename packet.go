package client

import "errors"

type packet struct {
	Type         packetType
	NSP          string
	Id           int
	Data         interface{}
	attachNumber int
}

type packetType int

//Const fields
const (
	_CONNECT packetType = iota
	_DISCONNECT
	_EVENT
	_ACK
	_ERROR
	_BINARY_EVENT
	_BINARY_ACK
	_CONNECTING
	_RECONNECT_FAILED
)

//Const fields
var (
	UnknowError = errors.New("unknow packet type.")
	EmptyString = ""
)

//String action to String
func (p packetType) String() (string, error) {
	switch p {
	case _CONNECT:
		return "connect", nil
	case _CONNECTING:
		return "connecting", nil
	case _DISCONNECT:
		return "disconnect", nil
	case _EVENT:
		return "event", nil
	case _ACK:
		return "ack", nil
	case _ERROR:
		return "error", nil
	case _BINARY_EVENT:
		return "binary_event", nil
	case _BINARY_ACK:
		return "binary_ack", nil
	case _RECONNECT_FAILED:
		return "reconnect_failed", nil
	}
	return EmptyString, UnknowError
}
