package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Message payload structure
type Message struct {
	Type     string `json:"type"`               // "text", "image", or "room_stats"
	RoomID   string `json:"room_id"`             // e.g. "room-alpha-123"
	Content  string `json:"content,omitempty"`  // Text snippet or base64 image string
	Browsers int    `json:"browsers,omitempty"` // Number of connected browser clients
	CLIs     int    `json:"clis,omitempty"`     // Number of connected CLI clients
}

// Room represents a stateful sync room in RAM
type Room struct {
	ID          string
	Clients     map[*websocket.Conn]string // Maps WebSocket connection to Client Type ("browser" or "cli")
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
			Clients: make(map[*websocket.Conn]string),
		}
		h.rooms[id] = room
	}
	return room
}

func (r *Room) BroadcastStats() {
	var browsers, clis int
	for _, cType := range r.Clients {
		if cType == "cli" {
			clis++
		} else {
			browsers++
		}
	}
	statsMsg := Message{
		Type:     "room_stats",
		RoomID:   r.ID,
		Browsers: browsers,
		CLIs:     clis,
	}
	for client := range r.Clients {
		_ = client.WriteJSON(statsMsg)
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins (Zero-Auth / CORS friendly)
	},
}

func main() {
	hub := NewHub()

	// WebSocket Endpoint: ws://localhost:8080/ws?room=room-123&client=browser|cli
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		roomID := r.URL.Query().Get("room")
		if roomID == "" {
			roomID = "default-room"
		}

		clientType := r.URL.Query().Get("client")
		if clientType != "cli" {
			clientType = "browser"
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("[Upgrade Error] %v", err)
			return
		}
		defer conn.Close()

		// Set max read limit to 20MB to support high-res base64 screenshots!
		conn.SetReadLimit(20 * 1024 * 1024)

		room := hub.GetRoom(roomID)

		// Register client connection
		room.mu.Lock()
		room.Clients[conn] = clientType

		// Push latest room state to the newly connected client
		if room.LatestText != "" {
			_ = conn.WriteJSON(Message{Type: "text", RoomID: roomID, Content: room.LatestText})
		}
		if room.LatestImage != "" {
			_ = conn.WriteJSON(Message{Type: "image", RoomID: roomID, Content: room.LatestImage})
		}

		// Broadcast connection stats to all clients in room
		room.BroadcastStats()
		room.mu.Unlock()

		go func() {
			time.Sleep(150 * time.Millisecond)
			room.mu.Lock()
			room.BroadcastStats()
			room.mu.Unlock()
		}()

		log.Printf("[Connected] %s joined Room: %s", clientType, roomID)

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

		// Unregister client on disconnect & broadcast stats update
		room.mu.Lock()
		delete(room.Clients, conn)
		room.BroadcastStats()
		room.mu.Unlock()
		log.Printf("[Disconnected] %s left Room: %s", clientType, roomID)
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
	fmt.Printf("   Port: %s | Zero-Auth | Connection Stats     \n", port)
	fmt.Printf("==================================================\n")
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
