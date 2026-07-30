package service

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/atotto/clipboard"
)

const IPCPort = "45731"

type DaemonState struct {
	PID       int       `json:"pid"`
	Mode      string    `json:"mode"` // "firebase" or "standalone"
	RoomID    string    `json:"room_id"`
	Port      string    `json:"port"`
	StartTime time.Time `json:"start_time"`
}

// MemoryLogHub holds in-memory logs and broadcasts live streams with ZERO disk files
type MemoryLogHub struct {
	mu        sync.Mutex
	history   []string
	listeners map[chan string]bool
}

var GlobalLogHub = &MemoryLogHub{
	history:   make([]string, 0, 200),
	listeners: make(map[chan string]bool),
}

// Write implements io.Writer to pipe log output to stdout AND in-memory stream
func (hub *MemoryLogHub) Write(p []byte) (n int, err error) {
	hub.mu.Lock()
	defer hub.mu.Unlock()

	line := string(p)
	// Maintain max 200 lines in memory
	if len(hub.history) >= 200 {
		hub.history = hub.history[1:]
	}
	hub.history = append(hub.history, line)

	// Broadcast to all active tailing HTTP streams
	for ch := range hub.listeners {
		select {
		case ch <- line:
		default:
		}
	}

	// Also print to stdout
	return os.Stdout.Write(p)
}

func (hub *MemoryLogHub) Subscribe() (chan string, []string) {
	hub.mu.Lock()
	defer hub.mu.Unlock()

	ch := make(chan string, 50)
	hub.listeners[ch] = true

	// Copy current in-memory history snapshot
	snapshot := make([]string, len(hub.history))
	copy(snapshot, hub.history)

	return ch, snapshot
}

func (hub *MemoryLogHub) Unsubscribe(ch chan string) {
	hub.mu.Lock()
	defer hub.mu.Unlock()

	delete(hub.listeners, ch)
	close(ch)
}

type IPCServer struct {
	state        DaemonState
	httpServer   *http.Server
	stopChan     chan struct{}
	onScreenshot func()
	onFetchText  func()
	onSendText   func()
}

func GetStateFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".ctrlv_state.json")
}

func SaveState(state DaemonState) error {
	filePath := GetStateFilePath()
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0644)
}

func LoadState() (*DaemonState, error) {
	filePath := GetStateFilePath()
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	var state DaemonState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func RemoveStateFile() {
	os.Remove(GetStateFilePath())
}

func NewIPCServer(mode string, roomID string, stopChan chan struct{}, onScreenshot func(), onFetchText func(), onSendText func()) *IPCServer {
	if mode == "" {
		mode = "firebase"
	}
	state := DaemonState{
		PID:       os.Getpid(),
		Mode:      mode,
		RoomID:    roomID,
		Port:      IPCPort,
		StartTime: time.Now(),
	}
	return &IPCServer{
		state:        state,
		stopChan:     stopChan,
		onScreenshot: onScreenshot,
		onFetchText:  onFetchText,
		onSendText:   onSendText,
	}
}

func (ipc *IPCServer) Start() error {
	if err := SaveState(ipc.state); err != nil {
		log.Printf("[IPC] Warning: Failed to save state file: %v", err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":     "running",
			"pid":        ipc.state.PID,
			"mode":       ipc.state.Mode,
			"room_id":    ipc.state.RoomID,
			"uptime":     time.Since(ipc.state.StartTime).String(),
			"start_time": ipc.state.StartTime.Format(time.RFC3339),
		})
	})

	// Trigger Screenshot Endpoint
	mux.HandleFunc("/snap", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if ipc.onScreenshot != nil {
			go ipc.onScreenshot()
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "message": "Screenshot triggered"})
	})

	// Trigger Fetch Endpoint
	mux.HandleFunc("/fetch", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if ipc.onFetchText != nil {
			go ipc.onFetchText()
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "message": "Fetch triggered"})
	})

	// Trigger Send Clipboard Text Question Endpoint
	mux.HandleFunc("/sendtext", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if ipc.onSendText != nil {
			go ipc.onSendText()
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "message": "Send text triggered"})
	})

	// Stop Endpoint
	mux.HandleFunc("/stop", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "message": "Stopping daemon"})
		if ipc.stopChan != nil {
			close(ipc.stopChan)
		}
	})

	// In-Memory Live Log Stream Endpoint (Zero Disk Files)
	mux.HandleFunc("/logs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		tail := r.URL.Query().Get("tail") == "true"
		ch, history := GlobalLogHub.Subscribe()
		defer GlobalLogHub.Unsubscribe(ch)

		flusher, ok := w.(http.Flusher)

		// Send historical logs first
		for _, line := range history {
			_, _ = fmt.Fprint(w, line)
		}
		if ok {
			flusher.Flush()
		}

		// If not tailing, finish request after history
		if !tail {
			return
		}

		// Continuously stream live logs
		for line := range ch {
			_, err := fmt.Fprint(w, line)
			if err != nil {
				break
			}
			if ok {
				flusher.Flush()
			}
		}
	})

	ipc.httpServer = &http.Server{
		Addr:    "127.0.0.1:" + IPCPort,
		Handler: mux,
	}

	go func() {
		if err := ipc.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[IPC] Server error: %v", err)
		}
	}()

	log.Printf("[IPC] Control server listening on 127.0.0.1:%s", IPCPort)
	return nil
}

func (ipc *IPCServer) Stop() {
	if ipc.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = ipc.httpServer.Shutdown(ctx)
	}
	RemoveStateFile()
}

// StandaloneSnap triggers silent screenshot capture via active daemon or fallback
func StandaloneSnap(credPath string, roomID string, quiet bool) error {
	// First try IPC if daemon is active
	state, err := LoadState()
	if err == nil && state.Port != "" {
		client := http.Client{Timeout: 2 * time.Second}
		resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%s/snap", state.Port))
		if err == nil {
			resp.Body.Close()
			if !quiet {
				fmt.Println("[ctrlv] Silent screen capture triggered via daemon!")
			}
			return nil
		}
	}

	// Fallback to direct standalone capture & Firestore upload
	if credPath == "" {
		return fmt.Errorf("serviceAccountKey.json not found")
	}

	b64Img, err := CaptureScreenSilent()
	if err != nil {
		return fmt.Errorf("failed to capture screen: %w", err)
	}

	fsService, err := NewFirestoreService(credPath)
	if err != nil {
		return fmt.Errorf("failed to connect to Firestore: %w", err)
	}
	defer fsService.Close()

	if err := fsService.UploadScreenshot(roomID, b64Img); err != nil {
		return fmt.Errorf("failed to upload screenshot: %w", err)
	}

	if !quiet {
		fmt.Println("[ctrlv] Silent screen capture uploaded to Firestore!")
	}
	return nil
}

// StandaloneFetch fetches text from Firestore and writes directly to system clipboard via xclip/wl-clipboard
func StandaloneFetch(credPath string, roomID string, quiet bool) error {
	// First try IPC if daemon is active
	state, err := LoadState()
	if err == nil && state.Port != "" {
		client := http.Client{Timeout: 2 * time.Second}
		resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%s/fetch", state.Port))
		if err == nil {
			resp.Body.Close()
			if !quiet {
				fmt.Println("[ctrlv] Text fetched & copied to clipboard via daemon!")
			}
			return nil
		}
	}

	// Fallback to direct standalone Firestore fetch & clipboard write
	if credPath == "" {
		return fmt.Errorf("serviceAccountKey.json not found")
	}

	fsService, err := NewFirestoreService(credPath)
	if err != nil {
		return fmt.Errorf("failed to connect to Firestore: %w", err)
	}
	defer fsService.Close()

	text, err := fsService.FetchTextAndMarkSeen(roomID)
	if err != nil {
		return fmt.Errorf("failed to fetch text: %w", err)
	}

	// Write directly to system clipboard (xclip / wl-clipboard on Linux)
	if err := clipboard.WriteAll(text); err != nil {
		return fmt.Errorf("failed to write text to system clipboard: %w", err)
	}

	if !quiet {
		fmt.Printf("[ctrlv] Text fetched & copied to system clipboard!\n")
	}
	return nil
}

// StandaloneSendText triggers reading PC clipboard text and uploading to Firestore or sending via daemon IPC
func StandaloneSendText(credPath string, roomID string, quiet bool) error {
	// First try IPC if daemon is active
	state, err := LoadState()
	if err == nil && state.Port != "" {
		client := http.Client{Timeout: 2 * time.Second}
		resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%s/sendtext", state.Port))
		if err == nil {
			resp.Body.Close()
			if !quiet {
				fmt.Println("[ctrlv] Clipboard text question sent via daemon!")
			}
			return nil
		}
	}

	// Fallback to direct standalone text upload
	clipText, err := clipboard.ReadAll()
	if err != nil || clipText == "" {
		return fmt.Errorf("clipboard is empty or unreadable")
	}

	if credPath == "" {
		return fmt.Errorf("serviceAccountKey.json not found")
	}

	fsService, err := NewFirestoreService(credPath)
	if err != nil {
		return fmt.Errorf("failed to connect to Firestore: %w", err)
	}
	defer fsService.Close()

	if err := fsService.UploadQuestionText(roomID, clipText); err != nil {
		return fmt.Errorf("failed to upload text question: %w", err)
	}

	if !quiet {
		fmt.Println("[ctrlv] Clipboard question text uploaded to Firestore!")
	}
	return nil
}

// QueryStatus queries the running background daemon
func QueryStatus() {
	state, err := LoadState()
	if err != nil {
		fmt.Println("ctrlv status: No background service running (state file not found).")
		return
	}

	resp, err := http.Get("http://127.0.0.1:" + state.Port + "/status")
	if err != nil {
		fmt.Printf("ctrlv status: Stopped or Unresponsive (PID %d recorded for room '%s').\n", state.PID, state.RoomID)
		RemoveStateFile()
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var statusData map[string]interface{}
	json.Unmarshal(body, &statusData)

	modeStr := statusData["mode"]
	if modeStr == nil || modeStr == "" {
		modeStr = "firebase"
	}

	fmt.Println("==================================================")
	fmt.Println("              ctrlv Daemon Status                 ")
	fmt.Println("==================================================")
	fmt.Printf(" Status:      %v\n", statusData["status"])
	fmt.Printf(" Mode:        %v\n", modeStr)
	fmt.Printf(" Process ID:  %v\n", statusData["pid"])
	if fmt.Sprintf("%v", modeStr) == "standalone" {
		fmt.Println(" Active Room: N/A (Standalone Direct AI Mode)")
	} else {
		fmt.Printf(" Active Room: %v\n", statusData["room_id"])
	}
	fmt.Printf(" Uptime:      %v\n", statusData["uptime"])
	fmt.Printf(" Started:     %v\n", statusData["start_time"])
	fmt.Println(" Hotkeys:     Ctrl+Shift+S (Screenshot) | Ctrl+Shift+T (Send Text) | Ctrl+Shift+F (Fetch)")
	fmt.Println(" CLI Triggers: ctrlv snap (Screenshot)  | ctrlv text (Send Text)   | ctrlv fetch (Fetch)")
	fmt.Println("==================================================")
}

// StreamLogs queries live in-memory logs via IPC HTTP stream (Zero Disk Files)
func StreamLogs(tail bool) {
	state, err := LoadState()
	if err != nil {
		fmt.Println("ctrlv logs: No active background daemon running.")
		return
	}

	url := fmt.Sprintf("http://127.0.0.1:%s/logs", state.Port)
	if tail {
		url += "?tail=true"
		fmt.Println("=== Tailing ctrlv Daemon In-Memory Logs (Press Ctrl+C to exit) ===")
	} else {
		fmt.Println("=== ctrlv Daemon In-Memory Logs ===")
	}

	client := http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Printf("Error creating log request: %v\n", err)
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("ctrlv logs: Failed to connect to running daemon log stream (PID %d).\n", state.PID)
		return
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		if line != "" {
			fmt.Print(line)
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			break
		}
	}
}

// RequestStop asks the running daemon to stop and terminates its process
func RequestStop() {
	state, err := LoadState()
	if err != nil {
		fmt.Println("ctrlv stop: No background service is currently running.")
		return
	}

	client := http.Client{Timeout: 1 * time.Second}
	resp, err := client.Post("http://127.0.0.1:"+state.Port+"/stop", "application/json", nil)
	if err == nil && resp != nil {
		resp.Body.Close()
	}

	// Kill PID recorded in state file to guarantee process termination
	if state.PID > 0 {
		proc, err := os.FindProcess(state.PID)
		if err == nil {
			_ = proc.Kill()
		}
	}

	RemoveStateFile()
	modeStr := state.Mode
	if modeStr == "" {
		modeStr = "firebase"
	}
	fmt.Printf("ctrlv stop: Background daemon (PID %d, Mode: %s) stopped successfully.\n", state.PID, modeStr)
}
