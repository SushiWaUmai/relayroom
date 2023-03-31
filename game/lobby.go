package game

import (
	"log"
	"net/http"
	"sync"

	"github.com/SushiWaUmai/game-relay-server/env"
	"github.com/gorilla/websocket"
)

type Lobby struct {
	JoinCode   string `json:"joinCode"`
	clients    map[*Client]bool
	join       chan *Client
	leave      chan *Client
	forward    chan []byte
	currentIdx int
}

var Lobbies sync.Map

func NewLobby() *Lobby {
	joincode := RandSeq(5)

	lobby := &Lobby{
		JoinCode: joincode,
		forward:  make(chan []byte),
		join:     make(chan *Client),
		leave:    make(chan *Client),
		clients:  make(map[*Client]bool),
	}

	Lobbies.Store(joincode, lobby)
	go lobby.Run()
	return lobby
}

func (l *Lobby) Run() {
	for {
		select {
		case client := <-l.join:
			l.clients[client] = true
		case client := <-l.leave:
			delete(l.clients, client)
			close(client.receive)
		case msg := <-l.forward:
			for client := range l.clients {
				client.receive <- msg
			}
		}
	}
}

func (l *Lobby) PlayerNum() int {
	return len(l.clients)
}

var upgrader = &websocket.Upgrader{ReadBufferSize: env.SOCKET_BUFFER_SIZE, WriteBufferSize: env.SOCKET_BUFFER_SIZE}

func (l *Lobby) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	socket, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Fatal("Failed to upgrade websocket connection", err)
		return
	}

	log.Println("Successfully upgraded websocket connection")

	client := &Client{
		Id:      l.currentIdx,
		socket:  socket,
		receive: make(chan []byte, env.MESSAGE_BUFFER_SIZE),
		lobby:   l,
	}
	l.currentIdx++

	l.join <- client
	defer func() {
		l.leave <- client
	}()
	go client.write()
	client.read()
}
