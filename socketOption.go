package client

//SocketOption options
type SocketOption struct {
	ReconnectionAttempts int
	ReconnectionDelay    int // how long to reconnect.  default value 5.
}
