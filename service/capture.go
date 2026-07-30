package service

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image/jpeg"
	"log"

	"github.com/kbinani/screenshot"
)

// CaptureScreenSilent captures the primary monitor silently and returns lightweight JPEG base64 string (~150KB)
func CaptureScreenSilent() (string, error) {
	n := screenshot.NumActiveDisplays()
	if n == 0 {
		return "", fmt.Errorf("no active displays available")
	}

	// Capture display 0 (primary screen)
	bounds := screenshot.GetDisplayBounds(0)
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return "", fmt.Errorf("failed to capture screen bounds %v: %w", bounds, err)
	}

	var buf bytes.Buffer
	// Encode as lightweight JPEG with 75% quality (reduces size by 90% from ~3.5MB PNG to ~180KB JPEG)
	opts := &jpeg.Options{Quality: 75}
	if err := jpeg.Encode(&buf, img, opts); err != nil {
		return "", fmt.Errorf("failed to encode image to JPEG: %w", err)
	}

	b64Data := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
	log.Printf("[Capture] Screen captured (%dx%d) compressed to JPEG (%d KB).", bounds.Dx(), bounds.Dy(), buf.Len()/1024)
	return b64Data, nil
}
