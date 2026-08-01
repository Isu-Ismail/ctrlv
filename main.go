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
	RoomID   string `json:"room_id"`             // e.g. "ctrlv-a8f3b2"
	Content  string `json:"content,omitempty"`  // Text snippet or base64 image string
	Browsers int    `json:"browsers,omitempty"` // Number of connected browser clients
	CLIs     int    `json:"clis,omitempty"`     // Number of connected CLI clients
	SenderID string `json:"sender_id,omitempty"` // Unique ID of sending client instance
}

// Room represents a stateful sync room in RAM
type Room struct {
	ID            string
	Clients       map[*websocket.Conn]string // Maps WebSocket connection to Client Type ("browser" or "cli")
	LatestWebText string
	LatestPCText  string
	LatestImage   string
	LastUpdated   time.Time
	mu            sync.RWMutex
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
			ID:          id,
			Clients:     make(map[*websocket.Conn]string),
			LastUpdated: time.Now(),
		}
		h.rooms[id] = room
	}
	return room
}

// Background worker to purge stale data and empty inactive rooms after 10-30 minutes
func (h *Hub) StartCleanupWorker() {
	log.Printf("[Init] Auto-Cleanup & RAM Purge Worker Active (5-min interval)")
	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		for range ticker.C {
			h.mu.Lock()
			now := time.Now()
			for id, room := range h.rooms {
				room.mu.Lock()
				if len(room.Clients) == 0 {
					// Clear text/image cache if empty for > 10 minutes
					if now.Sub(room.LastUpdated) > 10*time.Minute && (room.LatestWebText != "" || room.LatestPCText != "" || room.LatestImage != "") {
						room.LatestWebText = ""
						room.LatestPCText = ""
						room.LatestImage = ""
						log.Printf("[Cleanup] Wiped stale text & image RAM cache for empty Room: %s", id)
					}
					// Completely delete empty room from RAM if untouched for > 30 minutes
					if now.Sub(room.LastUpdated) > 30*time.Minute {
						delete(h.rooms, id)
						log.Printf("[Purge] Purged inactive Room from RAM: %s", id)
					}
				}
				room.mu.Unlock()
			}
			h.mu.Unlock()
		}
	}()
}

func (r *Room) BroadcastStats() {
	var deadConns []*websocket.Conn
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
		if err := client.WriteJSON(statsMsg); err != nil {
			deadConns = append(deadConns, client)
		}
	}
	if len(deadConns) > 0 {
		for _, dc := range deadConns {
			delete(r.Clients, dc)
		}
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins (Zero-Auth / CORS friendly)
	},
}

func main() {
	hub := NewHub()
	hub.StartCleanupWorker()

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
			return
		}
		defer conn.Close()

		// Max read payload limit (20MB for high-res base64 screenshots)
		conn.SetReadLimit(20 * 1024 * 1024)

		// 60-Second Dead Connection Timeout & Ping/Pong Heartbeat Setup
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		conn.SetPongHandler(func(string) error {
			conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			return nil
		})

		room := hub.GetRoom(roomID)

		// Register client connection
		room.mu.Lock()
		room.Clients[conn] = clientType
		room.LastUpdated = time.Now()

		// Push latest room state to the newly connected client
		if clientType == "cli" {
			if room.LatestWebText != "" {
				_ = conn.WriteJSON(Message{Type: "web_exe", RoomID: roomID, Content: room.LatestWebText})
			}
		} else {
			if room.LatestWebText != "" {
				_ = conn.WriteJSON(Message{Type: "web_exe", RoomID: roomID, Content: room.LatestWebText})
			}
			if room.LatestPCText != "" {
				_ = conn.WriteJSON(Message{Type: "exe_web", RoomID: roomID, Content: room.LatestPCText})
			}
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

		// Start Ping Heartbeat Ticker (Pings client every 25 seconds)
		pingTicker := time.NewTicker(25 * time.Second)
		defer pingTicker.Stop()

		go func() {
			for range pingTicker.C {
				if err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(5*time.Second)); err != nil {
					return
				}
			}
		}()

		// Read loop
		for {
			var msg Message
			err := conn.ReadJSON(&msg)
			if err != nil {
				break
			}
			msg.RoomID = roomID

			// Refresh connection deadline on active message
			conn.SetReadDeadline(time.Now().Add(60 * time.Second))

			// Save to state & broadcast to room
			room.mu.Lock()
			room.LastUpdated = time.Now()
			switch msg.Type {
			case "web_exe", "text":
				room.LatestWebText = msg.Content
			case "exe_web":
				room.LatestPCText = msg.Content
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
		room.LastUpdated = time.Now()
		room.BroadcastStats()
		room.mu.Unlock()
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
	fmt.Printf("   ctrlv Stateful WebSocket Relay Server v1.1.0  \n")
	fmt.Printf("   Port: %s | Zero-Auth | Auto-Cleanup & Purge   \n", port)
	fmt.Printf("==================================================\n")
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
