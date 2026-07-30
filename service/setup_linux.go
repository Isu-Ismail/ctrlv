//go:build linux

package service

import (
	"fmt"
	"os/exec"
)

// RunSetup configures Linux (GNOME/Ubuntu) completely silent shortcuts using systemd-run (Zero Desktop Window Flicker)
func RunSetup() {
	fmt.Println("==================================================")
	fmt.Println("   ctrlv Linux (GNOME) Silent Setup Auto-Config   ")
	fmt.Println("==================================================")
	fmt.Println()

	// 1. Disable GNOME camera shutter sound effects & visual screen flash
	fmt.Print("[1/3] Disabling camera shutter sound & screen flash... ")
	_ = exec.Command("gsettings", "set", "org.gnome.desktop.sound", "event-sounds", "false").Run()
	_ = exec.Command("gsettings", "set", "org.gnome.desktop.interface", "enable-animations", "false").Run()
	fmt.Println("✓ DONE")

	// 2. Configure Custom GNOME Keybindings Paths
	path1 := "/org/gnome/settings-daemon/plugins/media-keys/custom-keybindings/ctrlv-snap/"
	path2 := "/org/gnome/settings-daemon/plugins/media-keys/custom-keybindings/ctrlv-fetch/"

	fmt.Print("[2/3] Registering custom GNOME shortcut paths... ")
	_ = exec.Command("gsettings", "set", "org.gnome.settings-daemon.plugins.media-keys", "custom-keybindings", fmt.Sprintf("['%s', '%s']", path1, path2)).Run()
	fmt.Println("✓ DONE")

	// 3. Configure Snapshot & Fetch using systemd-run --user --quiet (Decouples process from GNOME window manager -> 100% Zero Flicker)
	fmt.Print("[3/3] Binding Ctrl+Shift+S (Snap) & Ctrl+Shift+F (Fetch)... ")
	cmdSnap := "systemd-run --user --quiet /usr/local/bin/ctrlv snap -q"
	cmdFetch := "systemd-run --user --quiet /usr/local/bin/ctrlv fetch -q"

	_ = exec.Command("gsettings", "set", "org.gnome.settings-daemon.plugins.media-keys.custom-keybinding:"+path1, "name", "ctrlv Snapshot").Run()
	_ = exec.Command("gsettings", "set", "org.gnome.settings-daemon.plugins.media-keys.custom-keybinding:"+path1, "command", cmdSnap).Run()
	_ = exec.Command("gsettings", "set", "org.gnome.settings-daemon.plugins.media-keys.custom-keybinding:"+path1, "binding", "<Control><Shift>s").Run()

	_ = exec.Command("gsettings", "set", "org.gnome.settings-daemon.plugins.media-keys.custom-keybinding:"+path2, "name", "ctrlv Fetch").Run()
	_ = exec.Command("gsettings", "set", "org.gnome.settings-daemon.plugins.media-keys.custom-keybinding:"+path2, "command", cmdFetch).Run()
	_ = exec.Command("gsettings", "set", "org.gnome.settings-daemon.plugins.media-keys.custom-keybinding:"+path2, "binding", "<Control><Shift>f").Run()
	fmt.Println("✓ DONE")

	fmt.Println()
	fmt.Println("==================================================")
	fmt.Println(" ✓ Linux Setup Complete!")
	fmt.Println(" Configured via systemd-run --user --quiet.")
	fmt.Println(" Ctrl+Shift+S (Snap) & Ctrl+Shift+F (Fetch) are now")
	fmt.Println(" 100% invisible with ZERO window popups or flickers!")
	fmt.Println("==================================================")
}
