package service

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type RelayMessage struct {
	Type    string `json:"type"`    // "text" or "image"
	RoomID  string `json:"room_id"`  // e.g. "room-alpha-123"
	Content string `json:"content"` // Text snippet or base64 image string
}

type RelayService struct {
	ServerURL string
	conn      *websocket.Conn
	mu        sync.Mutex
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
	return &RelayService{
		ServerURL: serverURL,
	}
}

func (rs *RelayService) Connect(roomID string) (*websocket.Conn, error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	if rs.conn != nil {
		_ = rs.conn.Close()
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

func (rs *RelayService) ListenRoomUpdates(ctx context.Context, roomID string, onNewText func(text string)) {
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

		log.Printf("[Relay] Connected to WebSocket room: %s", roomID)

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
				onNewText(msg.Content)
			}
		}

		time.Sleep(2 * time.Second)
	}
}

func (rs *RelayService) UploadScreenshot(roomID string, base64Image string) error {
	conn, err := rs.Connect(roomID)
	if err != nil {
		return err
	}
	defer func() {
		time.Sleep(250 * time.Millisecond)
		_ = conn.Close()
	}()

	msg := RelayMessage{
		Type:    "image",
		RoomID:  roomID,
		Content: base64Image,
	}

	if err := conn.WriteJSON(msg); err != nil {
		return fmt.Errorf("failed to send screenshot over relay: %w", err)
	}

	log.Printf("[Relay] Screenshot uploaded successfully to room: %s (%d KB)", roomID, len(base64Image)/1024)
	return nil
}

func (rs *RelayService) UploadQuestionText(roomID string, text string) error {
	conn, err := rs.Connect(roomID)
	if err != nil {
		return err
	}
	defer func() {
		time.Sleep(150 * time.Millisecond)
		_ = conn.Close()
	}()

	msg := RelayMessage{
		Type:    "text",
		RoomID:  roomID,
		Content: text,
	}

	if err := conn.WriteJSON(msg); err != nil {
		return fmt.Errorf("failed to send text over relay: %w", err)
	}

	log.Printf("[Relay] Question text uploaded successfully to room: %s", roomID)
	return nil
}

func (rs *RelayService) Close() {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	if rs.conn != nil {
		_ = rs.conn.Close()
		rs.conn = nil
	}
}
