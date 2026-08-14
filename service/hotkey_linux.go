//go:build linux

package service

import (
	"log"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
)

// Start implementation for Linux (Pure Go X11 Global Hotkey Listener)
func (h *HotkeyHandler) Start() {
	X, err := xgb.NewConn()
	if err != nil {
		log.Printf("[Hotkey Error] Linux X11 connection failed: %v. (Hotkeys available via CLI triggers)", err)
		<-h.stopChan
		return
	}
	defer X.Close()

	setup := xproto.Setup(X)
	screen := setup.DefaultScreen(X)
	root := screen.Root

	// Resolve keycodes for 'S', 'F', 'T', and 'H' dynamically
	minKC := setup.MinKeycode
	maxKC := setup.MaxKeycode
	mapping, err := xproto.GetKeyboardMapping(X, minKC, byte(maxKC-minKC+1)).Reply()
	if err != nil {
		log.Printf("[Hotkey Error] Failed to get X11 keyboard mapping: %v", err)
		<-h.stopChan
		return
	}

	var keycodeS, keycodeF, keycodeT, keycodeH xproto.Keycode
	keysymsPerKeycode := int(mapping.KeysymsPerKeycode)

	for i := 0; i < int(maxKC-minKC+1); i++ {
		kc := minKC + xproto.Keycode(i)
		for j := 0; j < keysymsPerKeycode; j++ {
			sym := mapping.Keysyms[i*keysymsPerKeycode+j]
			if (sym == 0x0073 || sym == 0x0053) && keycodeS == 0 { // 's' or 'S'
				keycodeS = kc
			}
			if (sym == 0x0066 || sym == 0x0046) && keycodeF == 0 { // 'f' or 'F'
				keycodeF = kc
			}
			if (sym == 0x0074 || sym == 0x0054) && keycodeT == 0 { // 't' or 'T'
				keycodeT = kc
			}
			if (sym == 0x0068 || sym == 0x0048) && keycodeH == 0 { // 'h' or 'H'
				keycodeH = kc
			}
		}
	}

	// Fallback keycodes if resolution failed (standard US QWERTY: S=39, F=41, T=28, H=43)
	if keycodeS == 0 {
		keycodeS = 39
	}
	if keycodeF == 0 {
		keycodeF = 41
	}
	if keycodeT == 0 {
		keycodeT = 28
	}
	if keycodeH == 0 {
		keycodeH = 43
	}

	// Modifiers: Ctrl + Alt (ModMaskControl | ModMask1)
	modCombo := uint16(xproto.ModMaskControl | xproto.ModMask1)
	modMasks := []uint16{
		modCombo,
		modCombo | uint16(xproto.ModMask2),
		modCombo | uint16(xproto.ModMaskLock),
		modCombo | uint16(xproto.ModMask2) | uint16(xproto.ModMaskLock),
	}

	// Grab Ctrl + Alt + S
	for _, m := range modMasks {
		_ = xproto.GrabKey(X, true, root, m, keycodeS, xproto.GrabModeAsync, xproto.GrabModeAsync)
	}

	// Grab Ctrl + Alt + F
	for _, m := range modMasks {
		_ = xproto.GrabKey(X, true, root, m, keycodeF, xproto.GrabModeAsync, xproto.GrabModeAsync)
	}

	// Grab Ctrl + Alt + T
	for _, m := range modMasks {
		_ = xproto.GrabKey(X, true, root, m, keycodeT, xproto.GrabModeAsync, xproto.GrabModeAsync)
	}

	// Grab Ctrl + Alt + H
	for _, m := range modMasks {
		_ = xproto.GrabKey(X, true, root, m, keycodeH, xproto.GrabModeAsync, xproto.GrabModeAsync)
	}

	log.Printf("[Hotkey] Linux X11 global hotkeys registered: Ctrl+Alt+S (Screenshot) | Ctrl+Alt+F (Fetch) | Ctrl+Alt+T (Send Clipboard) | Ctrl+Alt+H (Toggle Overlay)")

	// Event loop
	errChan := make(chan error, 1)
	go func() {
		for {
			ev, err := X.WaitForEvent()
			if err != nil {
				errChan <- err
				return
			}
			if ev == nil {
				continue
			}

			if kp, ok := ev.(xproto.KeyPressEvent); ok {
				switch kp.Detail {
				case keycodeS:
					log.Println("[Hotkey] Global hotkey triggered: Ctrl + Alt + S (Screenshot)")
					if h.onScreenshot != nil {
						go h.onScreenshot()
					}
				case keycodeF:
					log.Println("[Hotkey] Global hotkey triggered: Ctrl + Alt + F (Fetch Text)")
					if h.onFetchText != nil {
						go h.onFetchText()
					}
				case keycodeT:
					log.Println("[Hotkey] Global hotkey triggered: Ctrl + Alt + T (Send Clipboard Text)")
					if h.onSendText != nil {
						go h.onSendText()
					}
				case keycodeH:
					log.Println("[Hotkey] Global hotkey triggered: Ctrl + Alt + H (Toggle Overlay)")
					if h.onToggleOverlay != nil {
						go h.onToggleOverlay()
					}
				}
			}
		}
	}()

	select {
	case <-h.stopChan:
		return
	case err := <-errChan:
		log.Printf("[Hotkey Warning] X11 event loop stopped: %v", err)
	}
}

// Stop implementation for Linux
func (h *HotkeyHandler) Stop() {
	select {
	case h.stopChan <- struct{}{}:
	default:
	}
}
