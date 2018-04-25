package websocket

import (
	"net/http"
	"net/url"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestWebsocket(t *testing.T) {
	Convey("Creater", t, func() {
		So(Creater.Name, ShouldEqual, "websocket")
		So(Creater.Client, ShouldEqual, NewClient)
	})

	Convey("Normal work, client part", t, func() {
		addr := "http://localhost:5000"
		u, err := url.Parse(addr)
		So(err, ShouldBeNil)
		u.Scheme = "ws"
		req, err := http.NewRequest("GET", u.String(), nil)
		So(err, ShouldBeNil)
		So(req.URL.String(), ShouldEqual, "ws://localhost:5000")
		c, err := NewClient(req)
		So(err, ShouldBeNil)
		defer c.Close()
		// So(c.Response(), ShouldNotBeNil)
		// So(c.Response().StatusCode, ShouldEqual, http.StatusSwitchingProtocols)
		// {
		// 	w, err := c.NextWriter(message.MessageText, parser.MESSAGE)
		// 	So(err, ShouldBeNil)
		// 	_, err = w.Write([]byte("test"))
		// 	So(err, ShouldBeNil)
		// 	err = w.Close()
		// 	So(err, ShouldBeNil)
		// }
		// sync <- 1
		// <-sync
		// {
		// 	decoder, err := c.NextReader()
		// 	So(err, ShouldBeNil)
		// 	defer decoder.Close()
		// 	So(decoder.MessageType(), ShouldEqual, message.MessageText)
		// 	So(decoder.Type(), ShouldEqual, parser.OPEN)
		// 	b, err := ioutil.ReadAll(decoder)
		// 	So(err, ShouldBeNil)
		// 	So(string(b), ShouldEqual, "")
		// }
		// c.Close()
	})
}
