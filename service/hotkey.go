package service

type HotkeyHandler struct {
	onScreenshot func()
	onFetchText  func()
	onSendText   func()
	threadID     uint32
	stopChan     chan struct{}
}

func NewHotkeyHandler(onScreenshot func(), onFetchText func(), onSendText func()) *HotkeyHandler {
	return &HotkeyHandler{
		onScreenshot: onScreenshot,
		onFetchText:  onFetchText,
		onSendText:   onSendText,
		stopChan:     make(chan struct{}),
	}
}
