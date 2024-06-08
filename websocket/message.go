// message.go
package websocket

import "log"

type Message struct {
	Type       string `json:"type"`                  // 消息类型，例如 "notification"、"private"、"group"
	Content    string `json:"content"`               // 消息内容
	SenderID   string `json:"sender_id"`             // 发送者ID
	ReceiverID string `json:"receiver_id,omitempty"` // 接收者ID（私信）
	Room       string `json:"room,omitempty"`        // 群组ID（群聊）
	MessageID  string `json:"message_id,omitempty"`  // 消息ID（用于通知）
	URL        string `json:"url,omitempty"`
	FileType   string `json:"file_type,omitempty"`
}

func markAsRead(userID, messageID string) {
	// 在这里实现标记为已读的逻辑，例如更新数据库或缓存中的状态
	log.Printf("Marking message %s as read for user %s", messageID, userID)
}
