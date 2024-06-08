// main.go
package main

import (
	"gaozo-lib/websocket"
	"log"
	"net/http"
)

func main() {
	wsServer := websocket.NewWebSocketServer()
	go wsServer.Run()

	http.HandleFunc("/ws", wsServer.HandleConnections)

	log.Println("Starting server on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
