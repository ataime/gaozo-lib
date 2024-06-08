package websocket

import (
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var pongWait = 40 * time.Second
var pingPeriod = 30 * time.Second

type Manager struct {
	Clients    map[string]*Client          // 以用户ID为键的客户端映射
	Rooms      map[string]map[*Client]bool // 群组映射
	Broadcast  chan Message
	Register   chan *Client
	Unregister chan *Client
	Close      chan string
	mu         sync.Mutex
}

var manager = Manager{
	Clients:    make(map[string]*Client),
	Rooms:      make(map[string]map[*Client]bool),
	Broadcast:  make(chan Message),
	Register:   make(chan *Client),
	Unregister: make(chan *Client),
	Close:      make(chan string),
}

func (m *Manager) Start() {
	for {
		select {
		case client := <-m.Register:
			m.mu.Lock()
			m.Clients[client.UserID] = client
			m.mu.Unlock()
		case client := <-m.Unregister:
			m.mu.Lock()
			if _, ok := m.Clients[client.UserID]; ok {
				delete(m.Clients, client.UserID)
				close(client.Send)
			}

			m.mu.Unlock()
		case userID := <-m.Close: // 处理主动关闭连接
			m.mu.Lock()
			if client, ok := m.Clients[userID]; ok {
				client.Conn.Close()
				delete(m.Clients, userID)
				close(client.Send)
			}
			m.mu.Unlock()
		case message := <-m.Broadcast:
			m.mu.Lock()
			for _, client := range m.Clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(m.Clients, client.UserID)
				}
			}
			m.mu.Unlock()

		case message := <-m.Broadcast:
			switch message.Type {
			case "private":
				if client, ok := m.Clients[message.ReceiverID]; ok {
					client.Send <- message
				}
			case "group":
				if clients, ok := m.Rooms[message.Room]; ok {
					for client := range clients {
						client.Send <- message
					}
				}
			case "notification":
				if client, ok := m.Clients[message.ReceiverID]; ok {
					client.Send <- message
				}
			}
		}
	}
}

func init() {
	go manager.Start()
}

func GetClient(userID string) (*Client, bool) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	client, ok := manager.Clients[userID]
	return client, ok
}

func CloseClient(userID string) {
	manager.Close <- userID
}

func JoinGroup(userID, roomID string) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	client, ok := manager.Clients[userID]
	if !ok {
		return
	}
	if _, ok := manager.Rooms[roomID]; !ok {
		manager.Rooms[roomID] = make(map[*Client]bool)
	}
	manager.Rooms[roomID][client] = true
}

func LeaveGroup(userID, roomID string) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	client, ok := manager.Clients[userID]
	if !ok {
		return
	}
	if _, ok := manager.Rooms[roomID]; ok {
		delete(manager.Rooms[roomID], client)
		if len(manager.Rooms[roomID]) == 0 {
			delete(manager.Rooms, roomID)
		}
	}
}
