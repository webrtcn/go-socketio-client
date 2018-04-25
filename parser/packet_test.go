package parser

import (
	"bytes"
	"io"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestPacketType(t *testing.T) {
	Convey("Byte to type", t, func() {
		Convey("Open", func() {
			t, err := ByteToType(0)
			So(err, ShouldBeNil)
			So(t, ShouldEqual, OPEN)
		})
		Convey("Close", func() {
			t, err := ByteToType(1)
			So(err, ShouldBeNil)
			So(t, ShouldEqual, CLOSE)
		})
		Convey("Ping", func() {
			t, err := ByteToType(2)
			So(err, ShouldBeNil)
			So(t, ShouldEqual, PING)
		})
		Convey("Pong", func() {
			t, err := ByteToType(3)
			So(err, ShouldBeNil)
			So(t, ShouldEqual, PONG)
		})
		Convey("Message", func() {
			t, err := ByteToType(4)
			So(err, ShouldBeNil)
			So(t, ShouldEqual, MESSAGE)
		})
		Convey("Upgrade", func() {
			t, err := ByteToType(5)
			So(err, ShouldBeNil)
			So(t, ShouldEqual, UPGRADE)
		})
		Convey("Noop", func() {
			t, err := ByteToType(6)
			So(err, ShouldBeNil)
			So(t, ShouldEqual, NOOP)
		})
		Convey("Error", func() {
			_, err := ByteToType(7)
			So(err, ShouldNotBeNil)
		})
	})

	Convey("Type to byte", t, func() {
		Convey("Open", func() {
			So(OPEN.Byte(), ShouldEqual, 0)
		})
		Convey("Close", func() {
			So(CLOSE.Byte(), ShouldEqual, 1)
		})
		Convey("Ping", func() {
			So(PING.Byte(), ShouldEqual, 2)
		})
		Convey("Pong", func() {
			So(PONG.Byte(), ShouldEqual, 3)
		})
		Convey("Message", func() {
			So(MESSAGE.Byte(), ShouldEqual, 4)
		})
		Convey("Upgrade", func() {
			So(UPGRADE.Byte(), ShouldEqual, 5)
		})
		Convey("Noop", func() {
			So(NOOP.Byte(), ShouldEqual, 6)
		})
	})

}

func TestStringParser(t *testing.T) {
	type Test struct {
		name   string
		t      PacketType
		data   []byte
		output string
	}

	var tests = []Test{
		{"open data", OPEN, nil, "0"},
		{"message data", MESSAGE, []byte("test"), "4test"},
		{"ping data", PING, nil, "2"},
		{"pong data", PONG, nil, "3"},
		{"close data", CLOSE, nil, "1"},
	}

	for _, test := range tests {
		buf := bytes.NewBuffer(nil)
		Convey("Given a packet type "+test.name, t, func() {
			Convey("Create encoder", func() {
				encoder, err := NewStringEncoder(buf, test.t)
				So(err, ShouldBeNil)
				So(encoder, ShouldImplement, (*io.WriteCloser)(nil))
				Convey("Encoded", func() {
					for d := test.data; len(d) > 0; {
						n, err := encoder.Write(d)
						So(err, ShouldBeNil)
						d = d[n:]
					}
					Convey("End", func() {
						err := encoder.Close()
						So(err, ShouldBeNil)
						So(buf.String(), ShouldEqual, test.output)
					})
				})
			})
			Convey("Create decoder", func() {
				decoder, err := NewDecoder(buf)
				So(err, ShouldBeNil)
				So(decoder, ShouldImplement, (*io.ReadCloser)(nil))
				So(decoder.MessageType(), ShouldEqual, MessageText)
				Convey("Decoded", func() {
					So(decoder.Type(), ShouldEqual, test.t)
					decoded := make([]byte, len(test.data)+1)
					n, err := decoder.Read(decoded)
					if n > 0 {
						So(err, ShouldBeNil)
						So(decoded[:n], ShouldResemble, test.data)
					}
					Convey("End", func() {
						_, err := decoder.Read(decoded[:])
						So(err, ShouldEqual, io.EOF)
					})
				})
			})
		})
	}
}

func TestBinaryParser(t *testing.T) {
	type Test struct {
		name   string
		t      PacketType
		data   []byte
		output string
	}
	var tests = []Test{
		{"open data", OPEN, nil, "\x00"},
		{"message data", MESSAGE, []byte("test"), "\x04test"},
		{"ping data", PING, nil, "\x02"},
		{"pong data", PONG, nil, "\x03"},
		{"close data", CLOSE, nil, "\x01"},
	}
	for _, test := range tests {
		buf := bytes.NewBuffer(nil)
		Convey("Given a packet type "+test.name, t, func() {
			Convey("Create Encoder", func() {
				Convey("Create Encoder", func() {
					encoder, err := NewBinaryEncoder(buf, test.t)
					So(err, ShouldBeNil)
					So(encoder, ShouldImplement, (*io.WriteCloser)(nil))
					Convey("Encoded", func() {
						for d := test.data; len(d) > 0; {
							n, err := encoder.Write(d)
							So(err, ShouldBeNil)
							d = d[n:]
						}
						Convey("End", func() {
							err := encoder.Close()
							So(err, ShouldBeNil)
							So(buf.String(), ShouldEqual, test.output)
						})
					})
					Convey("Create decoder", func() {
						decoder, err := NewDecoder(buf)
						So(err, ShouldBeNil)
						So(decoder, ShouldImplement, (*io.ReadCloser)(nil))
						So(decoder.MessageType(), ShouldEqual, MessageBinary)

						Convey("Decoded", func() {
							So(decoder.Type(), ShouldEqual, test.t)
							decoded := make([]byte, len(test.data)+1)
							n, err := decoder.Read(decoded[:])
							if n > 0 {
								So(err, ShouldBeNil)
								//So(decoded[:n], ShouldResemble, test.data)
							}

							Convey("End", func() {
								_, err := decoder.Read(decoded[:])
								So(err, ShouldEqual, io.EOF)
							})
						})
					})

				})
			})
		})
	}
}

func TestBase64Parser(t *testing.T) {
	type Test struct {
		name   string
		t      PacketType
		data   []byte
		output string
	}
	var tests = []Test{
		{"open data", OPEN, nil, "b0"},
		{"message data", MESSAGE, []byte("test"), "b4dGVzdA=="},
	}
	for _, test := range tests {
		buf := bytes.NewBuffer(nil)
		Convey("Given a packet type "+test.name, t, func() {
			Convey("Create Encoder", func() {
				encoder, err := NewB64Encoder(buf, test.t)
				So(err, ShouldBeNil)
				So(encoder, ShouldImplement, (*io.WriteCloser)(nil))
				Convey("Encoded", func() {
					for d := test.data; len(d) > 0; {
						n, err := encoder.Write(d)
						So(err, ShouldBeNil)
						d = d[n:]
					}
					Convey("End", func() {
						err := encoder.Close()
						So(err, ShouldBeNil)
						So(buf.String(), ShouldEqual, test.output)
					})
				})
			})
			Convey("Create decoder", func() {
				decoder, err := NewDecoder(buf)
				So(err, ShouldBeNil)
				So(decoder, ShouldImplement, (*io.ReadCloser)(nil))
				So(decoder.MessageType(), ShouldEqual, MessageBinary)

				Convey("Decoded", func() {
					So(decoder.Type(), ShouldEqual, test.t)
					decoded := make([]byte, len(test.data)+1)
					n, err := decoder.Read(decoded[:])
					if n > 0 {
						So(err, ShouldBeNil)
						//So(decoded[:n], ShouldResemble, test.data)
					}

					Convey("End", func() {
						_, err := decoder.Read(decoded[:])
						if err != nil {
							So(err, ShouldEqual, io.EOF)
						}
					})
				})
			})
		})
	}
}

func TestLimitReaderDecoder(t *testing.T) {
	Convey("Test decoder with limit reader", t, func() {
		buf := bytes.NewBufferString("\x34\xe6\xb5\x8b\xe8\xaf\x95123")
		reader := newLimitReader(buf, 7)
		decoder, err := NewDecoder(reader)
		So(err, ShouldBeNil)
		So(decoder.Type(), ShouldEqual, MESSAGE)
		p := make([]byte, 6)
		decoder.Read(p)
		t := string(p)
		So(t, ShouldEqual, "测试")
		err = decoder.Close()
		So(err, ShouldBeNil)
		So(buf.String(), ShouldEqual, "123")
	})
}
