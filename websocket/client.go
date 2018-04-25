package websocket

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/webrtcn/go-socketio-client/parser"
	"github.com/webrtcn/go-socketio-client/transport"
)

const (
	protocol                = 3 //websocket version
	eioKey                  = "EIO"
	transportKey            = "transport"
	transportValue          = "websocket"
	webSocketProtocol       = "ws"
	webSocketSecureProtocol = "wss"
	httpProtocol            = "http"
	httpSecureProtocol      = "https"
	socketio                = "socket.io/"
)

type client struct {
	connection *websocket.Conn
	response   *http.Response
}

//NewClient create a new client instance.
func NewClient(req *http.Request) (transport.Client, error) {
	switch req.URL.Scheme {
	case httpProtocol:
		req.URL.Scheme = webSocketProtocol
	case httpSecureProtocol:
		req.URL.Scheme = webSocketSecureProtocol
	}
	if !strings.Contains(strings.ToLower(req.URL.Path), socketio) {
		req.URL.Path += socketio
	}
	querys := req.URL.Query()
	if v := querys.Get(eioKey); len(v) == 0 {
		querys.Add(eioKey, strconv.Itoa(protocol))
	}
	if v := querys.Get(transportKey); len(v) == 0 {
		querys.Add(transportKey, transportValue)
	}
	req.URL.RawQuery = querys.Encode()
	conn, resp, err := websocket.DefaultDialer.Dial(req.URL.String(), req.Header)
	if err != nil {
		return nil, err
	}
	return &client{
		connection: conn,
		response:   resp,
	}, nil
}

func (c *client) Response() *http.Response {
	return c.response
}

func (c *client) NextReader() (*parser.PacketDecoder, error) {
	var reader io.Reader
	for {
		t, r, err := c.connection.NextReader()
		if err != nil {
			return nil, err
		}
		switch t {
		case websocket.TextMessage:
			fallthrough
		case websocket.BinaryMessage:
			reader = r
			return parser.NewDecoder(reader)
		}
	}
}

func (c *client) NextWriter(msgType parser.MessageType, packetType parser.PacketType) (io.WriteCloser, error) {
	wsType, newEncoder := websocket.TextMessage, parser.NewStringEncoder
	if msgType == parser.MessageBinary {
		wsType, newEncoder = websocket.BinaryMessage, parser.NewBinaryEncoder
	}
	w, err := c.connection.NextWriter(wsType)
	if err != nil {
		return nil, err
	}
	ret, err := newEncoder(w, packetType)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (c *client) Close() error {
	return c.connection.Close()
}
