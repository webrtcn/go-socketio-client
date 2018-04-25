package websocket

import "github.com/webrtcn/go-socketio-client/transport"

//Creater return websocket creater
var Creater = transport.Creater{
	Name:      "websocket",
	Upgrading: true,
	Client:    NewClient,
}
