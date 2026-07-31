package service

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"log"

	"github.com/kbinani/screenshot"
)

// downsampleImage reduces resolution for high-DPI / 4K screens to make network sync lightning fast (<100ms)
func downsampleImage(src *image.RGBA, scale int) *image.RGBA {
	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	newW, newH := w/scale, h/scale
	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	for y := 0; y < newH; y++ {
		for x := 0; x < newW; x++ {
			dst.Set(x, y, src.At(x*scale, y*scale))
		}
	}
	return dst
}

// CaptureScreenSilent captures primary screen & compresses to ultra-lightweight JPEG (~60KB - 90KB) for instant sync
func CaptureScreenSilent() (string, error) {
	n := screenshot.NumActiveDisplays()
	if n == 0 {
		return "", fmt.Errorf("no active displays available")
	}

	bounds := screenshot.GetDisplayBounds(0)
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return "", fmt.Errorf("failed to capture screen bounds %v: %w", bounds, err)
	}

	var finalImg image.Image = img
	// Downsample high-DPI / 4K / 2K displays if width > 2000px
	if bounds.Dx() > 2000 {
		finalImg = downsampleImage(img, 2)
	}

	var buf bytes.Buffer
	// Encode as ultra-fast lightweight JPEG with 70% quality (~60KB)
	opts := &jpeg.Options{Quality: 70}
	if err := jpeg.Encode(&buf, finalImg, opts); err != nil {
		return "", fmt.Errorf("failed to encode image to JPEG: %w", err)
	}

	b64Data := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
	log.Printf("[Capture] Screen captured (%dx%d) optimized to %d KB.", bounds.Dx(), bounds.Dy(), buf.Len()/1024)
	return b64Data, nil
}
