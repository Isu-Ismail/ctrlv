package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/websocket"
)

// Message payload structure
type Message struct {
	Type    string `json:"type"`    // "text" or "image"
	RoomID  string `json:"room_id"`  // e.g. "room-alpha-123"
	Content string `json:"content"` // Text snippet or base64 image string
}

// Room represents a stateful sync room in RAM
type Room struct {
	ID          string
	Clients     map[*websocket.Conn]bool
	LatestText  string
	LatestImage string
	mu          sync.RWMutex
}

type Hub struct {
	rooms map[string]*Room
	mu    sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		rooms: make(map[string]*Room),
	}
}

func (h *Hub) GetRoom(id string) *Room {
	h.mu.Lock()
	defer h.mu.Unlock()

	room, exists := h.rooms[id]
	if !exists {
		room = &Room{
			ID:      id,
			Clients: make(map[*websocket.Conn]bool),
		}
		h.rooms[id] = room
	}
	return room
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins (Zero-Auth / CORS friendly)
	},
}

func main() {
	hub := NewHub()

	// WebSocket Endpoint: ws://localhost:8080/ws?room=room-123
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		roomID := r.URL.Query().Get("room")
		if roomID == "" {
			roomID = "default-room"
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("[Upgrade Error] %v", err)
			return
		}
		defer conn.Close()

		room := hub.GetRoom(roomID)

		// Register client connection
		room.mu.Lock()
		room.Clients[conn] = true

		// IMMEDIATELY push latest room state to the newly connected client!
		if room.LatestText != "" {
			_ = conn.WriteJSON(Message{Type: "text", RoomID: roomID, Content: room.LatestText})
		}
		if room.LatestImage != "" {
			_ = conn.WriteJSON(Message{Type: "image", RoomID: roomID, Content: room.LatestImage})
		}
		room.mu.Unlock()

		log.Printf("[Connected] Client joined Room: %s", roomID)

		// Read loop
		for {
			var msg Message
			err := conn.ReadJSON(&msg)
			if err != nil {
				break
			}
			msg.RoomID = roomID

			// Save to state & broadcast to room
			room.mu.Lock()
			switch msg.Type {
			case "text":
				room.LatestText = msg.Content
			case "image":
				room.LatestImage = msg.Content
			}

			for client := range room.Clients {
				_ = client.WriteJSON(msg)
			}
			room.mu.Unlock()
		}

		// Unregister client on disconnect
		room.mu.Lock()
		delete(room.Clients, conn)
		room.mu.Unlock()
		log.Printf("[Disconnected] Client left Room: %s", roomID)
	})

	// Health check endpoint for Cloud platforms (Render / Koyeb / Fly.io)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ctrlv relay server is healthy!")
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("==================================================\n")
	fmt.Printf("   ctrlv Stateful WebSocket Relay Server Running  \n")
	fmt.Printf("   Port: %s | Zero-Auth | Cross-Platform       \n", port)
	fmt.Printf("==================================================\n")
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
