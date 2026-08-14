//go:build windows

package service

import (
	"log"
	"runtime"
	"syscall"
	"unsafe"
)

// Win32 Hotkey Constants
const (
	MOD_ALT       = 0x0001
	MOD_CONTROL   = 0x0002
	MOD_SHIFT     = 0x0004
	MOD_WIN       = 0x0008
	MOD_NOREPEAT = 0x4000

	WM_HOTKEY = 0x0312
	WM_QUIT   = 0x0012

	VK_S = 0x53 // 'S' key
	VK_F = 0x46 // 'F' key
	VK_T = 0x54 // 'T' key
	VK_H = 0x48 // 'H' key

	HOTKEY_SCREENSHOT_ID   = 1001
	HOTKEY_FETCH_TEXT_ID   = 1002
	HOTKEY_SEND_TEXT_ID    = 1003
	HOTKEY_HIDE_OVERLAY_ID = 1004
)

type MSG struct {
	HWnd    syscall.Handle
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      struct{ X, Y int32 }
}

// Start registers Windows global hotkeys and listens in a dedicated event loop
func (h *HotkeyHandler) Start() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	user32 := syscall.NewLazyDLL("user32.dll")
	kernel32 := syscall.NewLazyDLL("kernel32.dll")

	procRegisterHotKey := user32.NewProc("RegisterHotKey")
	procUnregisterHotKey := user32.NewProc("UnregisterHotKey")
	procGetMessage := user32.NewProc("GetMessageW")
	procGetCurrentThreadId := kernel32.NewProc("GetCurrentThreadId")

	threadID, _, _ := procGetCurrentThreadId.Call()
	h.threadID = uint32(threadID)

	// Register Ctrl + Shift + S
	r1, _, err1 := procRegisterHotKey.Call(
		0,
		uintptr(HOTKEY_SCREENSHOT_ID),
		uintptr(MOD_CONTROL|MOD_SHIFT|MOD_NOREPEAT),
		uintptr(VK_S),
	)
	if r1 == 0 {
		log.Printf("[Hotkey] Warning: Failed to register Ctrl+Shift+S: %v", err1)
	} else {
		log.Println("[Hotkey] Global hotkey registered: Ctrl + Shift + S (Screenshot)")
	}

	// Register Ctrl + Shift + F
	r2, _, err2 := procRegisterHotKey.Call(
		0,
		uintptr(HOTKEY_FETCH_TEXT_ID),
		uintptr(MOD_CONTROL|MOD_SHIFT|MOD_NOREPEAT),
		uintptr(VK_F),
	)
	if r2 == 0 {
		log.Printf("[Hotkey] Warning: Failed to register Ctrl+Shift+F: %v", err2)
	} else {
		log.Println("[Hotkey] Global hotkey registered: Ctrl + Shift + F (Fetch Text)")
	}

	// Register Ctrl + Shift + T
	r3, _, err3 := procRegisterHotKey.Call(
		0,
		uintptr(HOTKEY_SEND_TEXT_ID),
		uintptr(MOD_CONTROL|MOD_SHIFT|MOD_NOREPEAT),
		uintptr(VK_T),
	)
	if r3 == 0 {
		log.Printf("[Hotkey] Warning: Failed to register Ctrl+Shift+T: %v", err3)
	} else {
		log.Println("[Hotkey] Global hotkey registered: Ctrl + Shift + T (Send Clipboard Text)")
	}

	// Register Ctrl + Shift + H (Toggle Overlay Hide/Show)
	r4, _, err4 := procRegisterHotKey.Call(
		0,
		uintptr(HOTKEY_HIDE_OVERLAY_ID),
		uintptr(MOD_CONTROL|MOD_SHIFT|MOD_NOREPEAT),
		uintptr(VK_H),
	)
	if r4 == 0 {
		log.Printf("[Hotkey] Warning: Failed to register Ctrl+Shift+H: %v", err4)
	} else {
		log.Println("[Hotkey] Global hotkey registered: Ctrl + Shift + H (Toggle Overlay Hide/Show)")
	}

	defer func() {
		procUnregisterHotKey.Call(0, uintptr(HOTKEY_SCREENSHOT_ID))
		procUnregisterHotKey.Call(0, uintptr(HOTKEY_FETCH_TEXT_ID))
		procUnregisterHotKey.Call(0, uintptr(HOTKEY_SEND_TEXT_ID))
		procUnregisterHotKey.Call(0, uintptr(HOTKEY_HIDE_OVERLAY_ID))
	}()

	var msg MSG
	for {
		ret, _, _ := procGetMessage.Call(
			uintptr(unsafe.Pointer(&msg)),
			0,
			0,
			0,
		)

		if int32(ret) <= 0 || msg.Message == WM_QUIT {
			log.Println("[Hotkey] Event loop terminated.")
			return
		}

		if msg.Message == WM_HOTKEY {
			switch msg.WParam {
			case HOTKEY_SCREENSHOT_ID:
				log.Println("[Hotkey] Pressed: Ctrl + Shift + S -> Capturing & Uploading Screenshot...")
				if h.onScreenshot != nil {
					go h.onScreenshot()
				}
			case HOTKEY_FETCH_TEXT_ID:
				log.Println("[Hotkey] Pressed: Ctrl + Shift + F -> Fetching Text & Updating Clipboard...")
				if h.onFetchText != nil {
					go h.onFetchText()
				}
			case HOTKEY_SEND_TEXT_ID:
				log.Println("[Hotkey] Pressed: Ctrl + Shift + T -> Reading Clipboard Text & Sending...")
				if h.onSendText != nil {
					go h.onSendText()
				}
			case HOTKEY_HIDE_OVERLAY_ID:
				log.Println("[Hotkey] Pressed: Ctrl + Shift + H -> Toggling Overlay Visibility...")
				if h.onToggleOverlay != nil {
					go h.onToggleOverlay()
				}
			}
		}
	}
}

// Stop posts WM_QUIT to terminate GetMessage loop on Windows
func (h *HotkeyHandler) Stop() {
	if h.threadID == 0 {
		return
	}
	user32 := syscall.NewLazyDLL("user32.dll")
	procPostThreadMessage := user32.NewProc("PostThreadMessageW")
	procPostThreadMessage.Call(uintptr(h.threadID), uintptr(WM_QUIT), 0, 0)
}
