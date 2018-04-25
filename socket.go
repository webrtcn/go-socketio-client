package client

import (
	"net/url"
	"reflect"
	"sync"
	"time"
)

//Const fields for On methods
const (
	OnConnection      = "connection"
	OnConnecting      = "connecting"
	OnDisConnection   = "disconnection"
	OnMessage         = "message"
	OnError           = "error"
	OnReconnectFailed = "reconnect_failed"
)

//Socket socket.io client for golang
type Socket struct {
	sessionID  string
	conn       *conn
	uri        *url.URL
	eventsLock sync.RWMutex
	events     map[string]*caller
	acks       map[int]*caller
	id         int
	namespace  string
	options    *SocketOption
	attempts   int
}

//Connect to socketio server
func Connect(uri string, options *SocketOption) (*Socket, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	if options == nil {
		options = &SocketOption{
			ReconnectionAttempts: 0,
			ReconnectionDelay:    5,
		}
	}
	c := &Socket{
		uri:     u,
		events:  make(map[string]*caller),
		acks:    make(map[int]*caller),
		options: options,
	}
	go c.connect()
	return c, nil
}

func (client *Socket) connect() {
	for {
		if client.conn != nil && client.conn.askForClosed {
			client.attempts = 0
			break
		}
		if client.options.ReconnectionAttempts > 0 {
			if client.attempts > client.options.ReconnectionAttempts {
				p := packet{
					Type: _RECONNECT_FAILED,
					Id:   -1,
				}
				client.onPacket(nil, &p)
				break
			} else {
				client.attempts++
			}
		}
		p := packet{
			Type: _CONNECTING,
			Id:   -1,
		}
		client.onPacket(nil, &p)
		socket, err := newConn(client.uri)
		if err != nil {
			if client.options.ReconnectionDelay <= 0 {
				client.options.ReconnectionDelay = 5
			}
			time.Sleep(time.Duration(client.options.ReconnectionDelay) * time.Second)
		} else {
			client.conn = socket
			client.attempts = 0
			go client.readLoop()
			break
		}
	}
}

//On get message from server
func (client *Socket) On(message string, fn interface{}) error {
	c, err := newCaller(fn)
	if err != nil {
		return err
	}
	client.eventsLock.Lock()
	client.events[message] = c
	client.eventsLock.Unlock()
	return err
}

//Emit send message to server
func (client *Socket) Emit(method string, args ...interface{}) error {
	var c *caller
	if l := len(args); l > 0 {
		fv := reflect.ValueOf(args[l-1])
		if fv.Kind() == reflect.Func {
			var err error
			c, err = newCaller(args[l-1])
			if err != nil {
				return err
			}
			args = args[:l-1]
		}
	}
	args = append([]interface{}{method}, args...)
	if c != nil {
		id, err := client.sendID(args)
		if err != nil {
			return err
		}
		client.acks[id] = c
		return nil
	}
	return client.send(args)
}

//GetSessionID get the current session id
func (client *Socket) GetSessionID() string {
	return client.sessionID
}

//Close close connection
func (client *Socket) Close() error {
	client.conn.askForClosed = true
	return client.conn.Close()
}

func (client *Socket) send(args []interface{}) error {
	packet := packet{
		Type: _EVENT,
		Id:   -1,
		NSP:  client.namespace,
		Data: args,
	}
	encoder := newEncoder(client.conn)
	return encoder.Encode(packet)
}

func (client *Socket) sendID(args []interface{}) (int, error) {
	packet := packet{
		Type: _EVENT,
		Id:   client.id,
		NSP:  client.namespace,
		Data: args,
	}
	client.id++
	if client.id < 0 {
		client.id = 0
	}
	encoder := newEncoder(client.conn)
	err := encoder.Encode(packet)
	if err != nil {
		return -1, nil
	}
	return packet.Id, nil
}

func (client *Socket) onAck(id int, decoder *decoder, packet *packet) error {
	c, ok := client.acks[id]
	if !ok {
		return nil
	}
	delete(client.acks, id)
	args := c.GetArgs()
	packet.Data = &args
	if err := decoder.DecodeData(packet); err != nil {
		return err
	}
	c.Call(args)
	return nil
}

func (client *Socket) onPacket(decoder *decoder, packet *packet) ([]interface{}, error) {
	var message string
	switch packet.Type {
	case _CONNECT:
		client.sessionID = client.conn.sessionid
		message = "connection"
	case _CONNECTING:
		message = "connecting"
	case _RECONNECT_FAILED:
		message = "reconnect_failed"
	case _DISCONNECT:
		message = "disconnection"
		go client.connect()
	case _ERROR:
		message = "error"
		go client.connect()
	case _ACK:
		fallthrough
	case _BINARY_ACK:
		return nil, client.onAck(packet.Id, decoder, packet)
	default:
		message = decoder.Message()
	}
	client.eventsLock.RLock()
	c, ok := client.events[message]
	client.eventsLock.RUnlock()
	if !ok {
		decoder.Close()
		return nil, nil
	}
	args := c.GetArgs()
	olen := len(args)
	if olen > 0 {
		packet.Data = &args
		if err := decoder.DecodeData(packet); err != nil {
			return nil, err
		}
	}
	for i := len(args); i < olen; i++ {
		args = append(args, nil)
	}

	retV := c.Call(args)
	if len(retV) == 0 {
		return nil, nil
	}

	var err error
	if last, ok := retV[len(retV)-1].Interface().(error); ok {
		err = last
		retV = retV[0 : len(retV)-1]
	}
	ret := make([]interface{}, len(retV))
	for i, v := range retV {
		ret[i] = v.Interface()
	}
	return ret, err
}

func (client *Socket) readLoop() error {
	defer func() {
		p := packet{
			Type: _DISCONNECT,
			Id:   -1,
		}
		client.onPacket(nil, &p)
	}()
	for {
		decoder := newDecoder(client.conn)
		var p packet
		if err := decoder.Decode(&p); err != nil {
			return err
		}
		ret, err := client.onPacket(decoder, &p)
		if err != nil {
			return err
		}
		switch p.Type {
		case _CONNECT:
			client.namespace = p.NSP
		case _BINARY_EVENT:
			fallthrough
		case _EVENT:
			if p.Id >= 0 {
				p := packet{
					Type: _ACK,
					Id:   p.Id,
					NSP:  client.namespace,
					Data: ret,
				}
				encoder := newEncoder(client.conn)
				if err := encoder.Encode(p); err != nil {
					return err
				}
			}
		case _DISCONNECT:
			return nil
		}
	}
}
