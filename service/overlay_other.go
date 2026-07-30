//go:build !windows

package service

import (
	"log"
)

// LaunchStealthOverlay implementation for non-Windows platforms
func LaunchStealthOverlay(roomID string) {
	log.Println("[Stealth Overlay] Screen-Share Protection Overlay launched for Linux/macOS...")
}

func UpdateOverlayText(text string) {
	// Fallback implementation
}

func UpdateOverlayStatus(statusMsg string) {
	// Fallback implementation
}
