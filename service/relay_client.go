package service

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type RelayMessage struct {
	Type     string `json:"type"`              // "text" or "image"
	RoomID   string `json:"room_id"`            // e.g. "room-alpha-123"
	Content  string `json:"content"`           // Text snippet or base64 image string
	SenderID string `json:"sender_id,omitempty"` // Unique ID of sending client instance
}

type RelayService struct {
	ServerURL  string
	InstanceID string
	conn       *websocket.Conn
	mu         sync.Mutex
	writeMu    sync.Mutex
}

const DefaultRelayURL = "wss://ctrlv.onrender.com/ws"

func GetRelayURL() string {
	cfg, err := LoadAppConfig()
	if err == nil && strings.TrimSpace(cfg.RelayURL) != "" {
		return strings.TrimSpace(cfg.RelayURL)
	}
	return DefaultRelayURL
}

func NewRelayService(serverURL string) *RelayService {
	if serverURL == "" {
		serverURL = GetRelayURL()
	}
	instID := fmt.Sprintf("cli-%d-%d", os.Getpid(), time.Now().UnixNano()%100000)
	return &RelayService{
		ServerURL:  serverURL,
		InstanceID: instID,
	}
}

func (rs *RelayService) Connect(roomID string) (*websocket.Conn, error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	if rs.conn != nil {
		_ = rs.conn.Close()
		rs.conn = nil
	}

	u, err := url.Parse(rs.ServerURL)
	if err != nil {
		return nil, fmt.Errorf("invalid relay server URL %s: %w", rs.ServerURL, err)
	}

	q := u.Query()
	q.Set("room", roomID)
	q.Set("client", "cli")
	u.RawQuery = q.Encode()

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to relay server at %s: %w", u.String(), err)
	}

	conn.SetReadLimit(20 * 1024 * 1024)
	rs.conn = conn
	return conn, nil
}

func (rs *RelayService) VerifyConnection(roomID string) error {
	conn, err := rs.Connect(roomID)
	if err != nil {
		return err
	}
	time.Sleep(100 * time.Millisecond)
	_ = conn.Close()
	return nil
}

func (rs *RelayService) ListenRoomUpdates(ctx context.Context, roomID string, onNewText func(text string, senderID string)) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		conn, err := rs.Connect(roomID)
		if err != nil {
			log.Printf("[Relay Connection Warning] %v", err)
			time.Sleep(3 * time.Second)
			continue
		}

		log.Printf("[Relay] Persistent CLI socket active (Instance: %s) for room: %s", rs.InstanceID, roomID)

		for {
			var msg RelayMessage
			err := conn.ReadJSON(&msg)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Printf("[Relay Read Warning] Connection lost: %v", err)
				break
			}

			if msg.Type == "text" && strings.TrimSpace(msg.Content) != "" {
				onNewText(msg.Content, msg.SenderID)
			}
		}

		time.Sleep(2 * time.Second)
	}
}

// SendMessage sends a payload over the active persistent socket, or opens a temporary one for one-shot commands
func (rs *RelayService) SendMessage(roomID string, msgType string, content string) error {
	rs.mu.Lock()
	activeConn := rs.conn
	rs.mu.Unlock()

	msg := RelayMessage{
		Type:     msgType,
		RoomID:   roomID,
		Content:  content,
		SenderID: rs.InstanceID,
	}

	// 1. If persistent daemon socket is connected, write over it directly! (No duplicate socket!)
	if activeConn != nil {
		rs.writeMu.Lock()
		err := activeConn.WriteJSON(msg)
		rs.writeMu.Unlock()
		if err == nil {
			log.Printf("[Relay] Pushed %s via persistent socket to room: %s (%d KB)", msgType, roomID, len(content)/1024)
			return nil
		}
		log.Printf("[Relay Warning] Write to active socket failed: %v. Reconnecting...", err)
	}

	// 2. Fallback for one-shot CLI commands (ctrlv snap -r room)
	tempConn, err := rs.Connect(roomID)
	if err != nil {
		return err
	}
	defer func() {
		time.Sleep(250 * time.Millisecond)
		_ = tempConn.Close()
	}()

	rs.writeMu.Lock()
	err = tempConn.WriteJSON(msg)
	rs.writeMu.Unlock()
	if err != nil {
		return fmt.Errorf("failed to send %s over relay: %w", msgType, err)
	}

	log.Printf("[Relay] Pushed %s via one-shot connection to room: %s (%d KB)", msgType, roomID, len(content)/1024)
	return nil
}

func (rs *RelayService) UploadScreenshot(roomID string, base64Image string) error {
	return rs.SendMessage(roomID, "image", base64Image)
}

func (rs *RelayService) UploadQuestionText(roomID string, text string) error {
	return rs.SendMessage(roomID, "text", text)
}

func (rs *RelayService) Close() {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	if rs.conn != nil {
		_ = rs.conn.Close()
		rs.conn = nil
	}
}
