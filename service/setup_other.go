//go:build !linux

package service

import (
	"fmt"
	"runtime"
)

// RunSetup implementation for non-Linux target platforms
func RunSetup() {
	fmt.Printf("Automated setup is not required for %s.\n", runtime.GOOS)
	fmt.Println("Global hotkeys are natively registered by the ctrlv daemon!")
}
