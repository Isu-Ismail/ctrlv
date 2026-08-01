//go:build windows

package service

import (
	"fmt"
	"log"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/atotto/clipboard"
)

var (
	modUser32                      = syscall.NewLazyDLL("user32.dll")
	procCreateWindowExW            = modUser32.NewProc("CreateWindowExW")
	procRegisterClassExW           = modUser32.NewProc("RegisterClassExW")
	procDefWindowProcW             = modUser32.NewProc("DefWindowProcW")
	procShowWindow                 = modUser32.NewProc("ShowWindow")
	procUpdateWindow               = modUser32.NewProc("UpdateWindow")
	procSetWindowPos               = modUser32.NewProc("SetWindowPos")
	procSetWindowDisplayAffinity    = modUser32.NewProc("SetWindowDisplayAffinity")
	procSetLayeredWindowAttributes = modUser32.NewProc("SetLayeredWindowAttributes")
	procSetWindowTextW             = modUser32.NewProc("SetWindowTextW")
	procGetWindowTextW             = modUser32.NewProc("GetWindowTextW")
	procSendMessageW               = modUser32.NewProc("SendMessageW")
	procGetMessageW                = modUser32.NewProc("GetMessageW")
	procTranslateMessage           = modUser32.NewProc("TranslateMessage")
	procDispatchMessageW           = modUser32.NewProc("DispatchMessageW")
	procPostQuitMessage            = modUser32.NewProc("PostQuitMessage")
	procMoveWindow                 = modUser32.NewProc("MoveWindow")
	procGetWindowRect              = modUser32.NewProc("GetWindowRect")
	procGetSystemMetrics           = modUser32.NewProc("GetSystemMetrics")
	procSetWindowRgn               = modUser32.NewProc("SetWindowRgn")

	modGdi32             = syscall.NewLazyDLL("gdi32.dll")
	procCreateFontW      = modGdi32.NewProc("CreateFontW")
	procCreateSolidBrush = modGdi32.NewProc("CreateSolidBrush")
	procSetTextColor     = modGdi32.NewProc("SetTextColor")
	procSetBkColor       = modGdi32.NewProc("SetBkColor")
	procSetBkMode        = modGdi32.NewProc("SetBkMode")
	procCreateRoundRectRgn = modGdi32.NewProc("CreateRoundRectRgn")
)

const (
	WS_EX_TOPMOST              = 0x00000008
	WS_EX_TOOLWINDOW           = 0x00000080
	WS_EX_LAYERED              = 0x00080000 // Translucent Glass Transparency!
	LWA_ALPHA                  = 0x00000002

	WS_POPUP                   = 0x80000000
	WS_VISIBLE                 = 0x10000000
	WS_CHILD                   = 0x40000000
	WS_CLIPCHILDREN            = 0x02000000
	WS_VSCROLL                 = 0x00200000
	WS_HSCROLL                 = 0x00100000
	ES_MULTILINE               = 0x0004
	ES_AUTOVSCROLL             = 0x0040
	ES_AUTOHSCROLL             = 0x0080
	ES_WANTRETURN              = 0x1000

	WDA_EXCLUDEFROMCAPTURE     = 0x00000011 // Hidden from Zoom, Teams, Meet, OBS, Discord, Screen Recordings!
	HWND_TOPMOST               = ^uintptr(0) // -1
	SWP_NOSIZE                 = 0x0001
	SWP_NOMOVE                 = 0x0002
	SWP_SHOWWINDOW             = 0x0040

	WM_DESTROY                 = 0x0002
	WM_SIZE                    = 0x0005
	WM_SETFONT                 = 0x0030
	WM_NCHITTEST               = 0x0084
	WM_COMMAND                 = 0x0111
	WM_CTLCOLOREDIT            = 0x0133
	WM_CTLCOLORSTATIC          = 0x0132
	EM_SETTABSTOPS             = 0x00CB

	HTCAPTION                  = 2
	HTLEFT                     = 10
	HTRIGHT                    = 11
	HTTOP                      = 12
	HTTOPLEFT                  = 13
	HTTOPRIGHT                 = 14
	HTBOTTOM                   = 15
	HTBOTTOMLEFT               = 16
	HTBOTTOMRIGHT              = 17

	SM_CXSCREEN                = 0
	ID_BTN_TOGGLE              = 101
	ID_EDIT_TEXT               = 102
	ID_STATIC_DIVIDER          = 103
	ID_BTN_CLEAR               = 104
	ID_BTN_COPY                = 105
	ID_TAB_0                   = 200
	ID_TAB_1                   = 201
	ID_TAB_2                   = 202
	ID_TAB_3                   = 203
	ID_TAB_4                   = 204
	ID_TAB_5                   = 205
)

type WNDCLASSEXW struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

type RECT struct {
	Left, Top, Right, Bottom int32
}

type RAMSnippetItem struct {
	text      string
	timestamp time.Time
}

var (
	overlayHWND        uintptr
	overlayEditHWND    uintptr
	overlayStatusHWND   uintptr
	overlayDividerHWND  uintptr
	overlayToggleBtn   uintptr
	overlayClearBtn    uintptr
	overlayCopyBtn     uintptr
	overlayTabBtns     [6]uintptr
	ramSnippets        []RAMSnippetItem = make([]RAMSnippetItem, 0, 6)
	activeRAMTab       int              = 0
	isCollapsed        bool
	roomIDGlobal       string
	fetchCount         int

	hMainBgBrush    uintptr
	hBtnBgBrush     uintptr
	hDividerBgBrush uintptr
	hEditBgBrush    uintptr

	expandedHeight  int32 = 320
	collapsedHeight int32 = 38
	overlayWidth    int32 = 640
	collapsedWidth  int32 = 100
)

func rgb(r, g, b byte) uint32 {
	return uint32(r) | (uint32(g) << 8) | (uint32(b) << 16)
}

func wndProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_NCHITTEST:
		ptX := int32(lParam & 0xffff)
		ptY := int32((lParam >> 16) & 0xffff)
		var rc RECT
		procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&rc)))

		// Border resizing detection when mouse cursor is within 8px of edges
		const bw int32 = 8

		left := ptX >= rc.Left && ptX < rc.Left+bw
		right := ptX <= rc.Right && ptX > rc.Right-bw
		top := ptY >= rc.Top && ptY < rc.Top+bw
		bottom := ptY <= rc.Bottom && ptY > rc.Bottom-bw

		if top && left {
			return HTTOPLEFT
		}
		if top && right {
			return HTTOPRIGHT
		}
		if bottom && left {
			return HTBOTTOMLEFT
		}
		if bottom && right {
			return HTBOTTOMRIGHT
		}
		if left {
			return HTLEFT
		}
		if right {
			return HTRIGHT
		}
		if top {
			return HTTOP
		}
		if bottom {
			return HTBOTTOM
		}

		// Top header area (38px height) for window dragging
		if ptY-rc.Top <= 38 {
			return HTCAPTION
		}
		return 1 // HTCLIENT

	case WM_CTLCOLOREDIT, WM_CTLCOLORSTATIC:
		hdc := wParam
		switch lParam {
		case overlayStatusHWND:
			// Top status header: Coral red / white text on dark glass
			procSetTextColor.Call(hdc, uintptr(rgb(248, 113, 113))) // Light Coral #F87171
			procSetBkColor.Call(hdc, uintptr(rgb(18, 18, 22)))     // Dark Translucent Glass #121216
			procSetBkMode.Call(hdc, 2)
			return hMainBgBrush
		case overlayToggleBtn, overlayClearBtn, overlayCopyBtn, overlayTabBtns[0], overlayTabBtns[1], overlayTabBtns[2], overlayTabBtns[3], overlayTabBtns[4], overlayTabBtns[5]:
			// Check if this is the active tab button to highlight it
			isActiveTab := false
			for i := 0; i < 6; i++ {
				if lParam == overlayTabBtns[i] && i == activeRAMTab {
					isActiveTab = true
					break
				}
			}
			if isActiveTab {
				// Highlight Active Tab in Emerald Green #10B981
				procSetTextColor.Call(hdc, uintptr(rgb(16, 185, 129)))
				procSetBkColor.Call(hdc, uintptr(rgb(30, 41, 59)))
			} else {
				// Styled Pill Buttons: Coral Red text on dark slate pill background
				procSetTextColor.Call(hdc, uintptr(rgb(239, 68, 68))) // Coral Red #EF4444
				procSetBkColor.Call(hdc, uintptr(rgb(30, 34, 43)))    // Dark Slate Button #1E222B
			}
			procSetBkMode.Call(hdc, 2)
			return hBtnBgBrush
		case overlayDividerHWND:
			// Subtle Grey Horizontal Divider
			procSetBkColor.Call(hdc, uintptr(rgb(51, 65, 85)))     // Dark Grey Divider #334155
			procSetBkMode.Call(hdc, 2)
			return hDividerBgBrush
		case overlayEditHWND:
			// Main Editor: Bright white text on dark editor background
			procSetTextColor.Call(hdc, uintptr(rgb(248, 250, 252))) // Bright White #F8FAFC
			procSetBkColor.Call(hdc, uintptr(rgb(22, 26, 35)))     // Dark Glass Editor #161A23
			procSetBkMode.Call(hdc, 2)
			return hEditBgBrush
		default:
			procSetTextColor.Call(hdc, uintptr(rgb(248, 250, 252)))
			procSetBkColor.Call(hdc, uintptr(rgb(18, 18, 22)))
			return hMainBgBrush
		}

	case WM_COMMAND:
		cmdID := uint16(wParam & 0xffff)
		switch cmdID {
		case ID_BTN_TOGGLE:
			toggleCollapse(hwnd)
		case ID_BTN_CLEAR:
			clearText()
		case ID_BTN_COPY:
			copyActiveTabText()
		case ID_TAB_0:
			selectRAMTab(0)
		case ID_TAB_1:
			selectRAMTab(1)
		case ID_TAB_2:
			selectRAMTab(2)
		case ID_TAB_3:
			selectRAMTab(3)
		case ID_TAB_4:
			selectRAMTab(4)
		case ID_TAB_5:
			selectRAMTab(5)
		}

	case WM_SIZE:
		width := int32(lParam & 0xffff)
		height := int32((lParam >> 16) & 0xffff)

		// Create smooth capsule rounded corners like iOS widget (24px corner radius)
		hRgn, _, _ := procCreateRoundRectRgn.Call(0, 0, uintptr(width), uintptr(height), 24, 24)
		if hRgn != 0 {
			procSetWindowRgn.Call(hwnd, hRgn, 1)
		}

		if isCollapsed {
			if overlayToggleBtn != 0 {
				procMoveWindow.Call(overlayToggleBtn, 8, 6, 84, 26, 1)
			}
			if overlayCopyBtn != 0 {
				procShowWindow.Call(overlayCopyBtn, 0)
			}
			for i := 0; i < 6; i++ {
				if overlayTabBtns[i] != 0 {
					procShowWindow.Call(overlayTabBtns[i], 0)
				}
			}
		} else {
			if overlayToggleBtn != 0 {
				procMoveWindow.Call(overlayToggleBtn, uintptr(width-75), 6, 65, 26, 1)
			}
			if overlayClearBtn != 0 {
				procMoveWindow.Call(overlayClearBtn, uintptr(width-145), 6, 62, 26, 1)
			}
			if overlayCopyBtn != 0 {
				procShowWindow.Call(overlayCopyBtn, 5)
				procMoveWindow.Call(overlayCopyBtn, uintptr(width-212), 6, 62, 26, 1)
			}
			if overlayStatusHWND != 0 {
				procMoveWindow.Call(overlayStatusHWND, 16, 8, uintptr(width-235), 24, 1)
			}
			if overlayDividerHWND != 0 {
				procMoveWindow.Call(overlayDividerHWND, 12, 36, uintptr(width-24), 1, 1)
			}
			startX := int32(12)
			for i := 0; i < 6; i++ {
				if overlayTabBtns[i] != 0 {
					w := int32(52)
					if i == 0 {
						w = 60
					}
					procShowWindow.Call(overlayTabBtns[i], 5)
					procMoveWindow.Call(overlayTabBtns[i], uintptr(startX), 42, uintptr(w), 24, 1)
					startX += w + 5
				}
			}
			if overlayEditHWND != 0 {
				procMoveWindow.Call(overlayEditHWND, 12, 72, uintptr(width-24), uintptr(height-84), 1)
			}
		}

	case WM_DESTROY:
		procPostQuitMessage.Call(0)
		return 0
	}

	res, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return res
}

func copyActiveTabText() {
	if activeRAMTab < len(ramSnippets) {
		activeText := ramSnippets[activeRAMTab].text
		if activeText != "" {
			if err := clipboard.WriteAll(activeText); err != nil {
				log.Printf("[Copy Error] %v", err)
			} else {
				if activeRAMTab == 0 {
					UpdateOverlayStatus("Copied Main Text to PC Clipboard!")
				} else {
					UpdateOverlayStatus(fmt.Sprintf("Copied Tab H%d to PC Clipboard!", activeRAMTab))
				}
			}
		}
	}
}

func clearText() {
	if overlayEditHWND == 0 {
		return
	}
	if activeRAMTab < len(ramSnippets) {
		ramSnippets = append(ramSnippets[:activeRAMTab], ramSnippets[activeRAMTab+1:]...)
	}
	refreshOverlayContent()
	UpdateOverlayStatus("Text Cleared")
}

func toggleCollapse(hwnd uintptr) {
	isCollapsed = !isCollapsed
	var rc RECT
	procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&rc)))
	x := rc.Left
	y := rc.Top

	if isCollapsed {
		// Completely hide header status, copy button, clear button, divider line, tab buttons, and text editor!
		procShowWindow.Call(overlayStatusHWND, 0)  // SW_HIDE
		procShowWindow.Call(overlayClearBtn, 0)   // SW_HIDE
		procShowWindow.Call(overlayCopyBtn, 0)    // SW_HIDE
		procShowWindow.Call(overlayDividerHWND, 0) // SW_HIDE
		procShowWindow.Call(overlayEditHWND, 0)    // SW_HIDE
		for i := 0; i < 6; i++ {
			if overlayTabBtns[i] != 0 {
				procShowWindow.Call(overlayTabBtns[i], 0) // SW_HIDE
			}
		}

		// Collapse window into mini floating pill containing only the Show + button
		procMoveWindow.Call(hwnd, uintptr(x), uintptr(y), uintptr(collapsedWidth), uintptr(collapsedHeight), 1)
		procMoveWindow.Call(overlayToggleBtn, 8, 6, 84, 26, 1)

		ptr, _ := syscall.UTF16PtrFromString("Show +")
		procSetWindowTextW.Call(overlayToggleBtn, uintptr(unsafe.Pointer(ptr)))
	} else {
		// Unhide all components
		procShowWindow.Call(overlayStatusHWND, 5)  // SW_SHOW
		procShowWindow.Call(overlayClearBtn, 5)   // SW_SHOW
		procShowWindow.Call(overlayCopyBtn, 5)    // SW_SHOW
		procShowWindow.Call(overlayDividerHWND, 5) // SW_SHOW
		procShowWindow.Call(overlayEditHWND, 5)    // SW_SHOW
		for i := 0; i < 6; i++ {
			if overlayTabBtns[i] != 0 {
				procShowWindow.Call(overlayTabBtns[i], 5) // SW_SHOW
			}
		}

		procMoveWindow.Call(hwnd, uintptr(x), uintptr(y), uintptr(overlayWidth), uintptr(expandedHeight), 1)
		procMoveWindow.Call(overlayToggleBtn, uintptr(overlayWidth-75), 6, 65, 26, 1)
		procMoveWindow.Call(overlayClearBtn, uintptr(overlayWidth-145), 6, 62, 26, 1)
		procMoveWindow.Call(overlayCopyBtn, uintptr(overlayWidth-212), 6, 62, 26, 1)

		ptr, _ := syscall.UTF16PtrFromString("Hide -")
		procSetWindowTextW.Call(overlayToggleBtn, uintptr(unsafe.Pointer(ptr)))
	}
}

func LaunchStealthOverlay(roomID string) {
	runtime.LockOSThread()
	roomIDGlobal = roomID

	// Create brushes for dark translucent glassmorphism widget design
	hMainBgBrush, _, _ = procCreateSolidBrush.Call(uintptr(rgb(18, 18, 22)))     // Deep Matte Black #121216
	hBtnBgBrush, _, _ = procCreateSolidBrush.Call(uintptr(rgb(30, 34, 43)))      // Styled Dark Button #1E222B
	hDividerBgBrush, _, _ = procCreateSolidBrush.Call(uintptr(rgb(51, 65, 85)))  // Dark Grey Divider #334155
	hEditBgBrush, _, _ = procCreateSolidBrush.Call(uintptr(rgb(22, 26, 35)))     // Dark Glass Editor #161A23

	className, _ := syscall.UTF16PtrFromString("CtrlVStealthOverlay")
	windowTitle, _ := syscall.UTF16PtrFromString("ctrlv Stealth Overlay")

	var wc WNDCLASSEXW
	wc.CbSize = uint32(unsafe.Sizeof(wc))
	wc.LpfnWndProc = syscall.NewCallback(wndProc)
	wc.LpszClassName = className
	wc.HbrBackground = hMainBgBrush

	procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))

	// Screen width for top-center positioning
	screenWidth, _, _ := procGetSystemMetrics.Call(SM_CXSCREEN)
	startX := (int32(screenWidth) - overlayWidth) / 2
	startY := int32(12)

	// Create Window with WS_EX_TOPMOST, WS_EX_TOOLWINDOW, WS_EX_LAYERED (Translucent Glass)
	hwnd, _, _ := procCreateWindowExW.Call(
		WS_EX_TOPMOST|WS_EX_TOOLWINDOW|WS_EX_LAYERED,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(windowTitle)),
		WS_POPUP|WS_VISIBLE|WS_CLIPCHILDREN,
		uintptr(startX), uintptr(startY),
		uintptr(overlayWidth), uintptr(expandedHeight),
		0, 0, 0, 0,
	)

	if hwnd == 0 {
		log.Println("[Stealth Overlay Error] Failed to create overlay window.")
		return
	}
	overlayHWND = hwnd

	// Set Translucent Glass Opacity (230 / 255 alpha for sleek dark transparency)
	procSetLayeredWindowAttributes.Call(hwnd, 0, 230, LWA_ALPHA)

	// Enable Screen-Share Protection (WDA_EXCLUDEFROMCAPTURE)
	// Window is 100% visible to user, but 100% INVISIBLE to Zoom, Teams, Meet, OBS, Discord, Screen Recordings!
	ret, _, _ := procSetWindowDisplayAffinity.Call(hwnd, WDA_EXCLUDEFROMCAPTURE)
	if ret != 0 {
		log.Println("[Stealth Overlay] Screen-Share Protection Active (WDA_EXCLUDEFROMCAPTURE set). Window is hidden from recordings & screen share!")
	} else {
		log.Println("[Stealth Overlay Warning] SetWindowDisplayAffinity returned 0.")
	}

	// Always on top
	procSetWindowPos.Call(hwnd, HWND_TOPMOST, 0, 0, 0, 0, SWP_NOMOVE|SWP_NOSIZE|SWP_SHOWWINDOW)

	// Top Status Header Label
	staticClass, _ := syscall.UTF16PtrFromString("STATIC")
	statusTitle, _ := syscall.UTF16PtrFromString(fmt.Sprintf("ctrlv [%s] • Ready (Ctrl+Shift+F)", roomID))
	statusHwnd, _, _ := procCreateWindowExW.Call(
		0, uintptr(unsafe.Pointer(staticClass)),
		uintptr(unsafe.Pointer(statusTitle)),
		WS_CHILD|WS_VISIBLE,
		16, 8, uintptr(overlayWidth-235), 24,
		hwnd, 0, 0, 0,
	)
	overlayStatusHWND = statusHwnd

	// Copy Pill Button
	btnClass, _ := syscall.UTF16PtrFromString("BUTTON")
	copyTextStr, _ := syscall.UTF16PtrFromString("Copy c")
	copyHwnd, _, _ := procCreateWindowExW.Call(
		0, uintptr(unsafe.Pointer(btnClass)),
		uintptr(unsafe.Pointer(copyTextStr)),
		WS_CHILD|WS_VISIBLE,
		uintptr(overlayWidth-212), 6, 62, 26,
		hwnd, uintptr(ID_BTN_COPY), 0, 0,
	)
	overlayCopyBtn = copyHwnd

	// Clear Button
	clearTextStr, _ := syscall.UTF16PtrFromString("Clear x")
	clearHwnd, _, _ := procCreateWindowExW.Call(
		0, uintptr(unsafe.Pointer(btnClass)),
		uintptr(unsafe.Pointer(clearTextStr)),
		WS_CHILD|WS_VISIBLE,
		uintptr(overlayWidth-145), 6, 62, 26,
		hwnd, uintptr(ID_BTN_CLEAR), 0, 0,
	)
	overlayClearBtn = clearHwnd

	// Styled Hide/Show Pill Button
	btnText, _ := syscall.UTF16PtrFromString("Hide -")
	btnHwnd, _, _ := procCreateWindowExW.Call(
		0, uintptr(unsafe.Pointer(btnClass)),
		uintptr(unsafe.Pointer(btnText)),
		WS_CHILD|WS_VISIBLE,
		uintptr(overlayWidth-75), 6, 65, 26,
		hwnd, uintptr(ID_BTN_TOGGLE), 0, 0,
	)
	overlayToggleBtn = btnHwnd

	// Subtle Grey Horizontal Divider Line
	divHwnd, _, _ := procCreateWindowExW.Call(
		0, uintptr(unsafe.Pointer(staticClass)),
		0,
		WS_CHILD|WS_VISIBLE,
		12, 36, uintptr(overlayWidth-24), 1,
		hwnd, uintptr(ID_STATIC_DIVIDER), 0, 0,
	)
	overlayDividerHWND = divHwnd

	// Font Setup for Header Label & Pill Buttons
	headerFontName, _ := syscall.UTF16PtrFromString("Segoe UI")
	hHeaderFont, _, _ := procCreateFontW.Call(
		14, 0, 0, 0, 600, 0, 0, 0, 0, 0, 0, 0, 0, uintptr(unsafe.Pointer(headerFontName)),
	)

	// Create 6 Tab Selection Pill Buttons: [Main] [H1] [H2] [H3] [H4] [H5]
	tabLabels := []string{"Main", "H1", "H2", "H3", "H4", "H5"}
	tabIDs := []uintptr{ID_TAB_0, ID_TAB_1, ID_TAB_2, ID_TAB_3, ID_TAB_4, ID_TAB_5}
	tabStartX := int32(12)
	for i := 0; i < 6; i++ {
		tPtr, _ := syscall.UTF16PtrFromString(tabLabels[i])
		w := int32(52)
		if i == 0 {
			w = 60
		}
		tHwnd, _, _ := procCreateWindowExW.Call(
			0, uintptr(unsafe.Pointer(btnClass)),
			uintptr(unsafe.Pointer(tPtr)),
			WS_CHILD|WS_VISIBLE,
			uintptr(tabStartX), 42, uintptr(w), 24,
			hwnd, tabIDs[i], 0, 0,
		)
		overlayTabBtns[i] = tHwnd
		if hHeaderFont != 0 {
			procSendMessageW.Call(tHwnd, WM_SETFONT, hHeaderFont, 1)
		}
		tabStartX += w + 5
	}

	// Multiline TextPad Edit Control with Autowrap & Multiline Tab Stop Support
	editClass, _ := syscall.UTF16PtrFromString("EDIT")
	editHwnd, _, _ := procCreateWindowExW.Call(
		0, uintptr(unsafe.Pointer(editClass)),
		0,
		WS_CHILD|WS_VISIBLE|WS_VSCROLL|WS_HSCROLL|ES_MULTILINE|ES_AUTOVSCROLL|ES_AUTOHSCROLL|ES_WANTRETURN,
		12, 72, uintptr(overlayWidth-24), uintptr(expandedHeight-84),
		hwnd, uintptr(ID_EDIT_TEXT), 0, 0,
	)
	overlayEditHWND = editHwnd

	// Monospace Font Setup for Code Indentation Preservation
	fontName, _ := syscall.UTF16PtrFromString("Consolas")
	hFont, _, _ := procCreateFontW.Call(
		16, 0, 0, 0, 400, 0, 0, 0, 0, 0, 0, 0, 0, uintptr(unsafe.Pointer(fontName)),
	)
	if hFont != 0 {
		procSendMessageW.Call(editHwnd, WM_SETFONT, hFont, 1)
	}

	if hHeaderFont != 0 {
		procSendMessageW.Call(statusHwnd, WM_SETFONT, hHeaderFont, 1)
		procSendMessageW.Call(btnHwnd, WM_SETFONT, hHeaderFont, 1)
		procSendMessageW.Call(clearHwnd, WM_SETFONT, hHeaderFont, 1)
		procSendMessageW.Call(copyHwnd, WM_SETFONT, hHeaderFont, 1)
	}

	// Set Tab Stops for clean indentations (4 spaces = 16 dialog units)
	tabStops := []int32{16}
	procSendMessageW.Call(editHwnd, EM_SETTABSTOPS, 1, uintptr(unsafe.Pointer(&tabStops[0])))

	procShowWindow.Call(hwnd, 5) // SW_SHOW
	procUpdateWindow.Call(hwnd)

	// Windows Message Loop
	var msg MSG
	for {
		res, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if res == 0 || int32(res) == -1 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

func selectRAMTab(idx int) {
	if idx < 0 || idx >= 6 {
		return
	}
	activeRAMTab = idx
	refreshOverlayContent()
}

func refreshOverlayContent() {
	if overlayEditHWND == 0 {
		return
	}
	var dispText string
	var timeStr string
	if activeRAMTab < len(ramSnippets) {
		dispText = ramSnippets[activeRAMTab].text
		timeStr = ramSnippets[activeRAMTab].timestamp.Format("15:04:05")
	} else {
		dispText = ""
	}

	ptr, err := syscall.UTF16PtrFromString(dispText)
	if err == nil {
		procSetWindowTextW.Call(overlayEditHWND, uintptr(unsafe.Pointer(ptr)))
	}

	for i := 0; i < 6; i++ {
		if overlayTabBtns[i] != 0 {
			var label string
			if i == 0 {
				label = "Main"
			} else {
				label = fmt.Sprintf("H%d", i)
			}
			if i == activeRAMTab {
				label = "[" + label + "]"
			}
			lPtr, _ := syscall.UTF16PtrFromString(label)
			procSetWindowTextW.Call(overlayTabBtns[i], uintptr(unsafe.Pointer(lPtr)))
		}
	}

	if activeRAMTab == 0 {
		if timeStr != "" {
			UpdateOverlayStatus(fmt.Sprintf("Main Text (%s)", timeStr))
		} else {
			UpdateOverlayStatus(fmt.Sprintf("Main Text (#%d in RAM)", len(ramSnippets)))
		}
	} else {
		if activeRAMTab < len(ramSnippets) {
			UpdateOverlayStatus(fmt.Sprintf("Viewing H%d (%s)", activeRAMTab, timeStr))
		} else {
			UpdateOverlayStatus(fmt.Sprintf("History H%d (Empty)", activeRAMTab))
		}
	}
}

func UpdateOverlayText(text string) {
	if overlayEditHWND == 0 {
		return
	}
	fetchCount++

	// Strip /---/ marker for clean overlay display
	cleanText := text
	if len(cleanText) > 6 && cleanText[len(cleanText)-6:] == "\n/---/" {
		cleanText = cleanText[:len(cleanText)-6]
	} else if len(cleanText) > 5 && cleanText[len(cleanText)-5:] == "/---/" {
		cleanText = cleanText[:len(cleanText)-5]
	}

	// Convert Unix \n to Win32 \r\n for perfect multiline indentation rendering
	cleanText = strings.ReplaceAll(cleanText, "\r\n", "\n")
	cleanText = strings.ReplaceAll(cleanText, "\n", "\r\n")

	// Push to top of RAM queue with timestamp (max 6 items)
	newItem := RAMSnippetItem{
		text:      cleanText,
		timestamp: time.Now(),
	}
	ramSnippets = append([]RAMSnippetItem{newItem}, ramSnippets...)
	if len(ramSnippets) > 6 {
		ramSnippets = ramSnippets[:6]
	}

	// Automatically switch to Main tab (0) on new text arrival
	activeRAMTab = 0
	refreshOverlayContent()
}

func UpdateOverlayStatus(statusMsg string) {
	if overlayStatusHWND == 0 {
		return
	}
	fullStatus := fmt.Sprintf("ctrlv [%s] • %s", roomIDGlobal, statusMsg)
	ptr, err := syscall.UTF16PtrFromString(fullStatus)
	if err == nil {
		procSetWindowTextW.Call(overlayStatusHWND, uintptr(unsafe.Pointer(ptr)))
	}
}
