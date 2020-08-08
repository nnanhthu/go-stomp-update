package main

import (
	"flag"
	"fmt"
	"go-stomp-update"

	//"signal"
	"os"

	//"github.com/drawdy/stomp-ws-go"
	"github.com/gorilla/websocket"
)

const defaultPort = ":61613"

var serverAddr = flag.String("server", "localhost:61613", "STOMP server endpoint")
var messageCount = flag.Int("count", 10, "Number of messages to send/receive")
var queueName = flag.String("queue", "/queue/client_test", "Destination queue")
var helpFlag = flag.Bool("help", false, "Print help text")
var stop = make(chan bool)

// these are the default options that work with RabbitMQ
var options []func(*stomp.Conn) error = []func(*stomp.Conn) error{
	stomp.ConnOpt.Login("guest", "guest"),
	stomp.ConnOpt.Host("/"),
}

func main() {
	flag.Parse()
	url := "wss://signal-controller-testing.quickom.com/signaling/classroom"
	token := "Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJxa2lkOm5vZGU6bWFzdGVyOmJhZmQ3N2I0LTE2MDQtNDc3Ni04NWI2LWRlMmJlNGIxNTcxZCIsInJvbGUiOiJub2RlIiwicXJDb2RlIjoiNzAxNTE1OTY3ODczNjQ5MjYiLCJleHAiOjE2Mjc1NTU3MjIsImlhdCI6MTU5Njc5NzMyMiwiZGV2aWNlSWQiOiJiYTAzZTYzMS0yYjczLTRlZDUtODI4NC02ZTExYjQ2ZTE4ZjgiLCJqdGkiOiJmNzZiZTRjMi02NjE3LTQ1NWMtYWQzYi1iMzNmYzZkZjcwMjMiLCJyb29tSWQiOiJyb29tOjcwMTUxNTk2Nzg3MzY0OTI2IiwiYXV0aG9yaXRpZXMiOlsiTk9ERSJdfQ.km8DYsKPN6dmqpOwgjgMkuMWKIsR4b97ud2_TiW7Ea8"
	//channel := "/classroom/room:70151596787364926"
	//dest := "/classroom/message/room:70151596787364926"
	//x := signal.NewSignaler(url, nil, token)
	//err := x.ConnectConn(channel)
	//url = "wss://192.168.1.248:8080/classroom/" //+ paddedRandomIntn(999) + "/" + uniuri.New() + "/websocket"
	url = "wss://signal-controller-testing.quickom.com/signaling/classroom/366/piwnsqkn/websocket"
	netConn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		println(err)
	}
	// Now create the stomp connection

	stompConn, err := stomp.Connect(netConn.UnderlyingConn(),
		stomp.ConnOpt.UseStomp,
		stomp.ConnOpt.Host(url),

		stomp.ConnOpt.Header("Authorization", token))

	if err != nil {
		println("cannot connect to server", err.Error())
	}
	println(stompConn)

	//err = x.Send(dest,"text/plain",[]byte("Hello"))
	println(err)
	if *helpFlag {
		fmt.Fprintf(os.Stderr, "Usage of %s\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	subscribed := make(chan bool)
	go recvMessages(subscribed)

	// wait until we know the receiver has subscribed
	<-subscribed

	go sendMessages()

	<-stop
	<-stop
}

func sendMessages() {
	defer func() {
		stop <- true
	}()

	conn, err := stomp.Dial("tcp", *serverAddr, options...)
	if err != nil {
		println("cannot connect to server", err.Error())
		return
	}

	for i := 1; i <= *messageCount; i++ {
		text := fmt.Sprintf("Message #%d", i)
		err = conn.Send(*queueName, "text/plain",
			[]byte(text), nil)
		if err != nil {
			println("failed to send to server", err)
			return
		}
	}
	println("sender finished")
}

func recvMessages(subscribed chan bool) {
	defer func() {
		stop <- true
	}()

	conn, err := stomp.Dial("tcp", *serverAddr, options...)

	if err != nil {
		println("cannot connect to server", err.Error())
		return
	}

	sub, err := conn.Subscribe(*queueName, stomp.AckAuto)
	if err != nil {
		println("cannot subscribe to", *queueName, err.Error())
		return
	}
	close(subscribed)

	for i := 1; i <= *messageCount; i++ {
		msg := <-sub.C
		expectedText := fmt.Sprintf("Message #%d", i)
		actualText := string(msg.Body)
		if expectedText != actualText {
			println("Expected:", expectedText)
			println("Actual:", actualText)
		}
	}
	println("receiver finished")

}
