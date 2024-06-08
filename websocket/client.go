// client.go
package websocket

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	Conn   *websocket.Conn
	UserID string
	Send   chan Message
	Groups map[string]bool // 加入的群组

	Role     string // 用户角色?
	Avatar   string // 用户头像?
	NickName string // 用户昵称?
}

func NewWebSocketClient(url string) (*Client, error) {
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, err
	}
	return &Client{Conn: conn}, nil
}

func (client *Client) SendMessage(message Message) error {
	return client.Conn.WriteJSON(message)
}

func (client *Client) ReceiveMessages() {
	for {
		var msg Message
		err := client.Conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("error: %v", err)
			client.Conn.Close()
			break
		}
		log.Printf("Received message: %+v", msg)
	}
}

func UpgradeToWebSocket(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	return upgrader.Upgrade(w, r, nil)
}

func InitializeClient(userID string, ws *websocket.Conn) *Client {
	client := &Client{
		Conn:   ws,
		UserID: userID,
		Send:   make(chan Message),
		Groups: make(map[string]bool),
	}
	manager.Register <- client
	go client.ReadPump()
	go client.WritePump()
	return client
}

// 在 WebSocket 连接中持续读取客户端发送的消息，并处理心跳机制和业务逻辑
func (c *Client) ReadPump() {
	defer func() {
		manager.Unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		var msg Message
		err := c.Conn.ReadJSON(&msg)
		if err != nil {
			log.Println("Read error:", err)
			break
		}
		switch msg.Type {
		case "mark_as_read":
			markAsRead(msg.SenderID, msg.MessageID)
		case "private", "group", "notification":
			manager.Broadcast <- msg
		}
	}
}

// 向客户端发送消息,每隔一段时间向客户端发送心跳
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.Send:
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteJSON(msg); err != nil {
				log.Println("Write error:", err)
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
