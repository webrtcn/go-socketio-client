package client

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/webrtcn/go-socketio-client/parser"
	"github.com/webrtcn/go-socketio-client/transport"
	"github.com/webrtcn/go-socketio-client/websocket"
)

var (
	defaultCreater = websocket.Creater
)

type state int

const (
	stateUnknow state = iota
	stateNormal
	stateClosing
	stateClosed
)

type conn struct {
	id              string
	url             *url.URL
	request         *http.Request
	writerLocker    sync.Mutex
	transportLocker sync.RWMutex
	currentName     string
	current         transport.Client
	state           state
	stateLocker     sync.RWMutex
	readerChan      chan *connReader
	sessionid       string
	pingTimeout     time.Duration
	pingInterval    time.Duration
	pingChan        chan bool
	askForClosed    bool
}

func newConn(url *url.URL) (*conn, error) {
	client := &conn{
		url:          url,
		state:        stateNormal,
		pingTimeout:  10 * time.Second,
		pingInterval: 5 * time.Second,
		pingChan:     make(chan bool),
		readerChan:   make(chan *connReader),
		askForClosed: false,
	}
	err := client.open()
	if err != nil {
		return nil, err
	}
	go client.readLoop()
	return client, nil
}

func (c *conn) ID() string {
	return c.id
}

func (c *conn) Request() *http.Request {
	return c.request
}

func (c *conn) NextReader() (parser.MessageType, io.ReadCloser, error) {
	if c.getState() == stateClosed {
		return parser.MessageBinary, nil, io.EOF
	}
	ret := <-c.readerChan
	if ret == nil {
		return parser.MessageBinary, nil, io.EOF
	}
	return parser.MessageType(ret.MessageType()), ret, nil
}

func (c *conn) NextWriter(t parser.MessageType) (io.WriteCloser, error) {
	switch c.getState() {
	case stateNormal:
	default:
		return nil, io.EOF
	}
	c.writerLocker.Lock()
	ret, err := c.getCurrent().NextWriter(parser.MessageType(t), parser.MESSAGE)
	if err != nil {
		c.writerLocker.Unlock()
		return ret, err
	}
	writer := newConnWriter(ret, &c.writerLocker)
	return writer, err
}

func (c *conn) Close() error {
	if c.getState() != stateNormal {
		return nil
	}
	c.writerLocker.Lock()
	if w, err := c.getCurrent().NextWriter(parser.MessageText, parser.CLOSE); err == nil {
		writer := newConnWriter(w, &c.writerLocker)
		writer.Close()
	} else {
		c.writerLocker.Unlock()
	}
	if err := c.getCurrent().Close(); err != nil {
		return err
	}
	c.setState(stateClosing)
	return nil
}

func (c *conn) OnPacket(r *parser.PacketDecoder) {
	if s := c.getState(); s != stateNormal {
		return
	}
	switch r.Type() {
	case parser.OPEN:
		var conninfo struct {
			SessionID    string `json:"sid"`
			PingTimeout  int    `json:"pingTimeout"`
			PingInterval int    `json:"pingInterval"`
		}
		b, _ := ioutil.ReadAll(r)
		defer func() {
			r.Close()
		}()
		err := json.Unmarshal(b, &conninfo)
		if err != nil { //get first message error. disconnect
			c.getCurrent().Close()
			return
		}
		c.sessionid = conninfo.SessionID
		c.pingInterval = time.Duration(conninfo.PingInterval/1000) * time.Second
		c.pingTimeout = time.Duration(conninfo.PingTimeout/1000) * time.Second
		go c.pingLoop()
	case parser.CLOSE:
		c.getCurrent().Close()
	case parser.PING:
		t := c.getCurrent()
		newWriter := t.NextWriter
		c.writerLocker.Lock()
		if w, _ := newWriter(parser.MessageText, parser.PONG); w != nil {
			io.Copy(w, r)
			w.Close()
		}
		c.writerLocker.Unlock()
		fallthrough
	case parser.PONG:
		c.pingChan <- true
	case parser.MESSAGE:
		closeChan := make(chan struct{})
		c.readerChan <- newConnReader(r, closeChan)
		<-closeChan
		close(closeChan)
		r.Close()
	}
}

func (c *conn) OnClose(server transport.Client) {
	t := c.getCurrent()
	if server != t {
		return
	}
	t.Close()
	c.setState(stateClosed)
	close(c.readerChan)
	close(c.pingChan)
}

func (c *conn) getState() state {
	c.stateLocker.RLock()
	defer c.stateLocker.RUnlock()
	return c.state
}

func (c *conn) setState(state state) {
	c.stateLocker.Lock()
	defer c.stateLocker.Unlock()
	c.state = state
}

func (c *conn) pingLoop() {
	lastPing := time.Now()
	lastTry := lastPing
	for {
		now := time.Now()
		pingDiff := now.Sub(lastPing)
		tryDiff := now.Sub(lastTry)
		afterPing := c.pingInterval - tryDiff
		afterTimeout := c.pingTimeout - pingDiff
		select {
		case ok := <-c.pingChan:
			if !ok {
				return
			}
			lastPing = time.Now()
			lastTry = lastPing
		case <-time.After(afterPing):
			c.writerLocker.Lock()
			if c.state != stateNormal {
				c.writerLocker.Unlock()
				return
			}
			if w, _ := c.getCurrent().NextWriter(parser.MessageText, parser.PING); w != nil {
				writer := newConnWriter(w, &c.writerLocker)
				writer.Close()
			} else {
				c.writerLocker.Unlock()
			}
			lastTry = time.Now()
		case <-time.After(afterTimeout):
			c.Close()
			return
		}
	}
}

func (c *conn) readLoop() {
	current := c.getCurrent()
	defer func() {
		c.OnClose(current)
	}()
	for {
		pack, err := current.NextReader()
		if err != nil {
			return
		}
		c.OnPacket(pack)
		pack.Close()
	}
}

func (c *conn) open() error {
	var err error
	c.request, err = http.NewRequest("GET", c.url.String(), nil)
	if err != nil {
		return err
	}
	t, err := defaultCreater.Client(c.request)
	if err != nil {
		return err
	}
	c.setCurrent("websocket", t)
	return nil
}

func (c *conn) setCurrent(name string, s transport.Client) {
	c.transportLocker.Lock()
	defer c.transportLocker.Unlock()
	c.currentName = name
	c.current = s
}

func (c *conn) getCurrent() transport.Client {
	c.transportLocker.RLock()
	defer c.transportLocker.RUnlock()
	return c.current
}
