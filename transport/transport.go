package transport

import (
	"io"
	"net/http"

	"github.com/webrtcn/go-socketio-client/parser"
)

//Callback callback function interface.
type Callback interface {
	OnPacket(r *parser.PacketDecoder)
}

//Creater get a transport instance.
type Creater struct {
	Name      string
	Upgrading bool
	Client    func(r *http.Request) (Client, error)
}

// Client is a transport layer in client to connect server.
type Client interface {

	//Response returns the response of last http request.
	Response() *http.Response

	//NextReader returns packet decoder. This function call should be synced.
	NextReader() (*parser.PacketDecoder, error)

	//NextWriter returns packet writer. This function call should be synced.
	NextWriter(messageType parser.MessageType, packetType parser.PacketType) (io.WriteCloser, error)

	//Close closes the transport.
	Close() error
}
