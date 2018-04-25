package client

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestClientConnect(t *testing.T) {
	Convey("Connect", t, func() {
		conn, err := Connect("http://localhost:3000")
		So(err, ShouldBeNil)
		So(conn, ShouldNotBeNil)
	})
}
