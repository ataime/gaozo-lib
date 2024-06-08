// server.go
package websocket

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

type WebSocketServer struct {
	Clients    map[*websocket.Conn]bool
	Broadcast  chan Message
	Register   chan *websocket.Conn
	Unregister chan *websocket.Conn
	Rooms      map[string]*Room
}

func NewWebSocketServer() *WebSocketServer {
	return &WebSocketServer{
		Clients:    make(map[*websocket.Conn]bool),
		Broadcast:  make(chan Message),
		Register:   make(chan *websocket.Conn),
		Unregister: make(chan *websocket.Conn),
	}
}

func (server *WebSocketServer) Run() {
	for {
		select {
		case conn := <-server.Register:
			server.Clients[conn] = true
		case conn := <-server.Unregister:
			if _, ok := server.Clients[conn]; ok {
				delete(server.Clients, conn)
				conn.Close()
			}
		case message := <-server.Broadcast:
			for client := range server.Clients {
				err := client.WriteJSON(message)
				if err != nil {
					log.Printf("error: %v", err)
					client.Close()
					delete(server.Clients, client)
				}
			}
		}
	}
}

func (server *WebSocketServer) HandleConnections(w http.ResponseWriter, r *http.Request) {
	conn, err := Upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatalf("Could not upgrade connection: %v", err)
		return
	}
	server.Register <- conn

	for {
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("error: %v", err)
			server.Unregister <- conn
			break
		}
		server.Broadcast <- msg
	}
}

func (server *WebSocketServer) GetRoom(name string) *Room {
	room, exists := server.Rooms[name]
	if !exists {
		room = NewRoom(name)
		server.Rooms[name] = room
		go room.RunRoom()
	}
	return room
}
