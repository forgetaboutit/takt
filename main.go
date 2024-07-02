package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"sync"
)

func main() {
	var hostFlag = flag.String("host", "localhost", "the host to use")
	var portFlag = flag.Int("port", 9999, "the port to use")
	flag.Parse()

	serverErr := becomeServer(*portFlag)

	if serverErr != nil {
		clientErr := becomeClient(*hostFlag, *portFlag)

		if clientErr != nil {
			log.Fatal("Couldn't start either server or client: ", clientErr)
		}
	}
}

func becomeServer(port int) error {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))

	if err != nil {
		return err
	}

	defer l.Close()
	log.Println("Listening on port", port, "...")

	server := Server{
		clients:       make(map[net.Addr]*Client),
		broadcastChan: make(chan Message, 100),
		RWMutex:       sync.RWMutex{},
	}

	server.startBroadcasting()

	for {
		conn, err := l.Accept()

		if err != nil {
			log.Println("Error accepting connection", err)
		}

		go handleConnection(&server, conn)
	}
}

func becomeClient(host string, port int) error {
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	conn, err := net.Dial("tcp", addr)

	if err != nil {
		return err
	}

	defer conn.Close()

	_ = conn

	return nil
}

type Server struct {
	clients       map[net.Addr]*Client
	broadcastChan chan Message
	sync.RWMutex
}

func (server *Server) add(client *Client) {
	server.Lock()
	defer server.Unlock()
	server.clients[client.RemoteAddr()] = client
}

func (server *Server) remove(client *Client) {
	server.Lock()
	defer server.Unlock()
	delete(server.clients, client.RemoteAddr())
}

func (server *Server) broadcastMessage(msg Message) {
	server.RLock()
	defer server.RUnlock()

	for _, client := range server.clients {
		client.send(msg)
	}
}

func (server *Server) startBroadcasting() {
	go func() {
		for {
			msg := <-server.broadcastChan
			server.broadcastMessage(msg)
		}
	}()
}

func (server *Server) broadcast(message Message) {
	logger := log.Println

	if message.text[len(message.text)-1:] == "\n" {
		logger = log.Print
	}

	logger("Received message: ", message.text)

	server.broadcastChan <- message
}

type Client struct {
	*Server
	net.Conn
	sendChan chan<- Message
}

func (client Client) send(message Message) {
	messageBytes := []byte(message.text)
	writtenBytes, err := client.Write(messageBytes)

	if err != nil {
		log.Println("Error writing to client:", err)
		return
	}

	if writtenBytes < len(messageBytes) {
		log.Println("Message was ", len(messageBytes), " bytes, send only ", writtenBytes)
	}
}

type Message struct {
	text string
}

func handleConnection(server *Server, conn net.Conn) {
	client := Client{
		Server:   server,
		Conn:     conn,
		sendChan: make(chan Message, 10),
	}

	server.add(&client)
	defer server.remove(&client)

	log.Println("Client connected:", conn.RemoteAddr())

	recvBuffer := make([]byte, 1024*4)

	for {
		bytesRead, err := conn.Read(recvBuffer)

		if err != nil {
			log.Println("Error reading from connection, disconnecting:", err)

			conn.Close()
			return
		}

		message := string(recvBuffer[:bytesRead])
		server.broadcast(Message{
			text: message,
		})
	}
}
