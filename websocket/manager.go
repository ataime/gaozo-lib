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
	Groups     map[string]map[*Client]bool // 群组映射
	Broadcast  chan Message
	Register   chan *Client
	Unregister chan *Client
	Close      chan string
	mu         sync.Mutex
}

var manager = Manager{
	Clients:    make(map[string]*Client),
	Groups:     make(map[string]map[*Client]bool),
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
			for groupID := range client.Groups {
				if _, ok := m.Groups[groupID]; ok {
					delete(m.Groups[groupID], client)
					if len(m.Groups[groupID]) == 0 {
						delete(m.Groups, groupID)
					}
				}
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
				if clients, ok := m.Groups[message.GroupID]; ok {
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

func JoinGroup(userID, groupID string) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	client, ok := manager.Clients[userID]
	if !ok {
		return
	}
	if _, ok := manager.Groups[groupID]; !ok {
		manager.Groups[groupID] = make(map[*Client]bool)
	}
	manager.Groups[groupID][client] = true
	client.Groups[groupID] = true
}

func LeaveGroup(userID, groupID string) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	client, ok := manager.Clients[userID]
	if !ok {
		return
	}
	if _, ok := manager.Groups[groupID]; ok {
		delete(manager.Groups[groupID], client)
		if len(manager.Groups[groupID]) == 0 {
			delete(manager.Groups, groupID)
		}
	}
	delete(client.Groups, groupID)
}
