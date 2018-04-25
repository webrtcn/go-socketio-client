
# go-socket.io-client

golang implementation of socket.io-client library 

#### Installation


```
go get github.com/webrtcn/go-socketio-client
```



#### Example

```
package main

import (
	"fmt"
	socket "github.com/webrtcn/go-socketio-client"
)

func main() {
	go func() {
		options := &socket.SocketOption{
			ReconnectionDelay:    3,
			ReconnectionAttempts: 10,
		}
		s, err := socket.Connect("http://example.com", options)
		if err != nil {
			return
		}
		s.On(socket.OnConnection, func() {
			fmt.Println("Connect to server successful.")
			fmt.Println(s.GetSessionID())    //sid
			s.Emit("message", "hello word.") // send string message
			var data struct {
				Title   string
				Message string
				Type    int
			}
			data.Title = "test"
			data.Message = "This is a test message"
			data.Type = 1
			s.Emit(socket.OnMessage, data)                  //send struct message
			s.Emit("message", "welcome", func(msg string) { // send with ack message
				fmt.Println(msg)
			})
		})
		s.On(socket.OnMessage, func(msg string) string { //listen with ack message
			fmt.Println(msg)
			return "yes"
		})
		s.On(socket.OnConnecting, func() {
			fmt.Println("connecting to server")
		})
		s.On(socket.OnReconnectFailed, func() {
			fmt.Println("connect to server failed")
		})
		s.On(socket.OnDisConnection, func() {
			fmt.Println("server disconnect.")
		})
		if err != nil {
			fmt.Println(err)
		}
	}()
	select {}
} 
```
