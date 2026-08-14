package service

type HotkeyHandler struct {
	onScreenshot    func()
	onFetchText     func()
	onSendText      func()
	onToggleOverlay func()
	threadID        uint32
	stopChan        chan struct{}
}

func NewHotkeyHandler(onScreenshot func(), onFetchText func(), onSendText func(), onToggleOverlay func()) *HotkeyHandler {
	return &HotkeyHandler{
		onScreenshot:    onScreenshot,
		onFetchText:     onFetchText,
		onSendText:      onSendText,
		onToggleOverlay: onToggleOverlay,
		stopChan:        make(chan struct{}),
	}
}
