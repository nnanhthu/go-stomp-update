package main

import (
	"flag"
	_ "flag"
	_ "fmt"
	"go-stomp-update"
	"math/rand"
	"strconv"
	"strings"
)

var channel = flag.String("channel", "/classroom/room:70151596787364926", "Channel")
var messageDest = flag.String("messageDest", "/classroom/room:70151596787364926", "Message Destination")
var authorization = flag.String("authorization", "Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJxa2lkOm5vZGU6bWFzdGVyOmJhZmQ3N2I0LTE2MDQtNDc3Ni04NWI2LWRlMmJlNGIxNTcxZCIsInJvbGUiOiJub2RlIiwicXJDb2RlIjoiNzAxNTE1OTY3ODczNjQ5MjYiLCJleHAiOjE2Mjc1NTU3MjIsImlhdCI6MTU5Njc5NzMyMiwiZGV2aWNlSWQiOiJiYTAzZTYzMS0yYjczLTRlZDUtODI4NC02ZTExYjQ2ZTE4ZjgiLCJqdGkiOiJmNzZiZTRjMi02NjE3LTQ1NWMtYWQzYi1iMzNmYzZkZjcwMjMiLCJyb29tSWQiOiJyb29tOjcwMTUxNTk2Nzg3MzY0OTI2IiwiYXV0aG9yaXRpZXMiOlsiTk9ERSJdfQ.km8DYsKPN6dmqpOwgjgMkuMWKIsR4b97ud2_TiW7Ea8", "Stomp token")

var options = []func(*stomp.Conn) error{
	stomp.ConnOpt.AcceptVersion(stomp.V12),
	stomp.ConnOpt.Header("Authorization", *authorization),
}

func paddedRandomInt(max int) string {
	var (
		ml = len(strconv.Itoa(max))
		ri = rand.Intn(max)
		is = strconv.Itoa(ri)
	)

	if len(is) < ml {
		is = strings.Repeat("0", ml-len(is)) + is
	}

	return is
}

type MessageData struct {
	Text string `json:"text"`
}

type Message struct {
	Name string       `json:"name"`
	Data *MessageData `json:"data"`
}

//func main() {
//	println("Hello World")
//
//	url := "wss://signal-controller-testing.quickom.com/signaling/classroom/" + paddedRandomInt(999) + "/" + uniuri.New() + "/websocket"
//	//url := "ws://192.168.1.248:8080/classroom/" + paddedRandomIntn(999) + "/" + uniuri.New() + "/websocket"
//	//url := "ws://localhost:8080/classroom/" + paddedRandomIntn(999) + "/" + uniuri.New() + "/websocket"
//
//	// Open WebSocket
//	println("Opening WebSocket...")
//	netConn, _, err := websocket.DefaultDialer.Dial(url, nil)
//	if err != nil {
//		println("cannot connect to server", err.Error())
//		return
//	}
//	println("WebSocket Opened!!\n")
//
//	// Connect Stomp
//	println("Connecting Stomp...")
//	stompConn, err := stomp.Connect(netConn, options...)
//
//	if err != nil {
//		println("cannot connect to server", err.Error())
//		return
//	}
//
//	println("Stomp Connected!!\n")
//
//	println("Subscribing Stomp Channel...")
//	sub, err := stompConn.Subscribe(*channel, stomp.AckAuto)
//	if err != nil {
//		println("cannot subscribe to", *channel, err.Error())
//		return
//	}
//
//	println("Stomp Channel Subscribed!!!\n")
//
//	println("Sending hello message...")
//	text := "{\"name\":\"HELLO_WORLD\"}"
//	err = stompConn.Send(*messageDest, "application/json", []byte(text), nil)
//	if err != nil {
//		println("failed to send to server", err)
//		return
//	}
//	println("Hello Message Sent!!!\n")
//
//	// process incoming message
//	go processLoop(sub)
//
//
//	// process send message from console
//	reader := bufio.NewReader(os.Stdin)
//	println("Simple Shell")
//	println("Type a text to send to stomp")
//	println("Type \"exit\" to disconnect stomp")
//	println("---------------------\n")
//
//	for {
//		fmt.Print("-> ")
//		text, _ := reader.ReadString('\n')
//		// convert CRLF to LF
//		text = strings.Replace(text, "\n", "", -1)
//
//		if strings.Compare("exit", text) == 0 {
//			stompConn.Disconnect()
//			os.Exit(1)
//		}
//
//		if len(text) <= 0 {
//			continue
//		}
//
//		data := &MessageData{
//			Text: text,
//		}
//		message := &Message{
//			Name: "HELLO_WORLD",
//			Data: data,
//		}
//		b, err := json.Marshal(message)
//		if err != nil {
//			println("error", err)
//		}
//
//		println("sending message: ", string(b))
//		err = stompConn.Send(*messageDest, "application/json", b, nil)
//		if err != nil {
//			println("failed to send to server", err)
//		}
//		println("sent!!!")
//	}
//}

func processLoop(sub *stomp.Subscription) {
	for {
		msg, err := sub.Read()
		if err != nil {
			println("Read error", err)
			continue
		}
		actualText := string(msg.Body)
		println("Receive message:", actualText)
	}
}
