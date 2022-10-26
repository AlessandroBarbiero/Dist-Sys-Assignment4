package main

import (
	"bufio"
	"chat/chat"
	"context"
	"log"
	"os"
	"strconv"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	id          int64
	vectorClock map[int64]int64
	mutex       sync.RWMutex
}

// Add this part if we want to use parametric port on command line
// var serverPort = flag.Int("serverPort", 8080, "server port number")  //name, port, usage OK

func main() {
	// Get the port from the command line when the client is run
	// flag.Parse()

	// Hardcoded port
	port_value := 8080
	serverPort := &port_value

	client := &Client{
		id:          1,
		vectorClock: make(map[int64]int64),
		mutex:       sync.RWMutex{},
	}

	client.startClient(serverPort)
}

func (c *Client) startClient(serverPort *int) {
	serverConnection := getServerConnection(serverPort)

	ctx := context.Background()
	stream, err := serverConnection.Chat(ctx)
	if err != nil {
		log.Fatalln("Client couldn't connect")
	}

	msg, err := stream.Recv()
	if err != nil {
		log.Fatalln("Couldn't get id from server")
	}

	c.id = msg.Id
	c.vectorClock[c.id] = 0

	go c.receiveTHD(stream)

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		input := scanner.Text()
		c.sendMessage(stream, input)
		log.Printf("Message sent:\"%s\"\n", input)
	}
}

func (c *Client) receiveTHD(stream chat.Chat_ChatClient) {
	for {
		msg, err := stream.Recv()
		if err != nil {
			log.Fatal("Disrupted connection")
		}

		log.Printf("Received message from %d: %s send at %v", msg.Id, msg.Message, msg.VectorClock)
		c.mergeVectorClocks(msg.VectorClock)
		c.updateMyTime()
		log.Printf("Current node time is %v", c.vectorClock)
	}
}

func (c *Client) updateMyTime() map[int64]int64 {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.vectorClock[c.id]++

	return c.vectorClock
}

func Max(x, y int64) int64 {
	if x < y {
		return y
	}
	return x
}

func (c *Client) mergeVectorClocks(clock map[int64]int64) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for client, clientTime := range clock {
		_, ok := c.vectorClock[client]
		if ok {
			c.vectorClock[client] = Max(c.vectorClock[client], clientTime)
		} else {
			c.vectorClock[client] = clientTime
		}
	}
}

// Update the vector clock and send a message with the given content and the updated clock
func (c *Client) sendMessage(stream chat.Chat_ChatClient, content string) {

	clock := c.updateMyTime()

	err := stream.SendMsg(&chat.RequestMsg{
		Id:          int64(c.id),
		Message:     content,
		VectorClock: clock,
	})

	if err != nil {
		log.Printf("Couldn't send message")
	}
}

func getServerConnection(serverPort *int) chat.ChatClient {

	// Run server and client on local host if you don't write ip address
	conn, err := grpc.Dial(":"+strconv.Itoa(*serverPort), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalln("could not dial")
	}
	log.Printf("Dialed\n")

	return chat.NewChatClient(conn)
}
