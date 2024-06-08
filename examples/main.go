// main.go
package main

import (
	"log"
	"net/http"

	"github.com/ataime/gaozo-lib/websocket"
	"github.com/gin-gonic/gin"
)

func main() {
	websocket.InitScyllaDB()
	defer websocket.CloseScyllaDB()

	r := gin.Default()
	wsServer := websocket.NewWebSocketServer()
	go wsServer.Run()

	http.HandleFunc("/ws", wsServer.HandleConnections)
	sslCert := "ssl/server.crt"
	sslKey := "ssl/server.key"

	log.Println("Starting server on :8080")
	if err := http.ListenAndServeTLS(":8080", sslCert, sslKey, r); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
