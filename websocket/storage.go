package websocket

import (
	"fmt"
	"log"

	"github.com/gocql/gocql"
)

var session *gocql.Session

func InitScyllaDB() {
	cluster := gocql.NewCluster("localhost")
	cluster.Keyspace = "chat"
	cluster.Consistency = gocql.Quorum

	var err error
	session, err = cluster.CreateSession()
	if err != nil {
		log.Fatalf("Failed to connect to ScyllaDB: %v", err)
	}
}

func SaveMessage(msg Message) error {
	if session == nil {
		return fmt.Errorf("ScyllaDB session is not initialized")
	}

	query := `INSERT INTO messages (id, type, room, sender, content, url, file_type) VALUES (?, ?, ?, ?, ?, ?, ?)`
	err := session.Query(query, gocql.TimeUUID(), msg.Type, msg.Room, msg.SenderID, msg.Content, msg.URL, msg.FileType).Exec()
	return err
}

func CloseScyllaDB() {
	if session != nil {
		session.Close()
	}
}
