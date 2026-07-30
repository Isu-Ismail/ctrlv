//go:build !windows && !linux

package service

import (
	"log"
)

// Start implementation for other platforms (macOS / BSD)
func (h *HotkeyHandler) Start() {
	log.Println("[Hotkey] Global hotkeys listener started for other OS platform...")
	<-h.stopChan
}

// Stop implementation for other platforms
func (h *HotkeyHandler) Stop() {
	select {
	case h.stopChan <- struct{}{}:
	default:
	}
}
