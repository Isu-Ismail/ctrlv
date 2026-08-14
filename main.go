package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"ctrlv/service"

	"github.com/atotto/clipboard"
)

const Version = "v1.0.0"

func printVersion() {
	fmt.Printf("ctrlv %s (Realtime Cross-Device Screenshot & Clipboard Sync CLI)\n", Version)
}

func main() {
	// Custom usage output (prevents Go standard flag library from dumping auto-generated flags)
	flag.Usage = printUsage

	// Check raw command line args for version or help flags BEFORE flag.Parse()
	for _, arg := range os.Args[1:] {
		if arg == "-v" || arg == "--version" || arg == "version" {
			printVersion()
			return
		}
		if arg == "-h" || arg == "--help" || arg == "help" || arg == "-help" {
			printUsage()
			return
		}
	}

	// Parse Flags & Subcommands
	isDaemonWorker := flag.Bool("daemon", false, "Internal worker daemon flag")
	roomFlag := flag.String("r", "", "Room ID to connect")
	screenFlagShort := flag.Bool("s", false, "Launch stealth screen-share invisible overlay window")
	screenFlagLong := flag.Bool("screen", false, "Launch stealth screen-share invisible overlay window")
	flag.Parse()

	args := flag.Args()

	// Ensure ctrlv_config.json is auto-created on ANY command run
	_, _, _ = service.EnsureConfigExists()

	// Check if subcommands were passed
	if len(args) > 0 {
		switch args[0] {
		case "config":
			editorFlag := ""
			for i, a := range args[1:] {
				if (a == "-e" || a == "--editor") && i+1 < len(args[1:]) {
					editorFlag = args[1:][i+1]
				} else if strings.HasPrefix(a, "-e=") {
					editorFlag = strings.TrimPrefix(a, "-e=")
				} else if strings.HasPrefix(a, "--editor=") {
					editorFlag = strings.TrimPrefix(a, "--editor=")
				}
			}
			if err := service.OpenConfigInEditor(editorFlag); err != nil {
				fmt.Printf("[Error] Failed to open config editor: %v\n", err)
			}
			return
		case "setup":
			service.RunSetup()
			return
		case "status":
			service.QueryStatus()
			return
		case "stop":
			service.RequestStop()
			return
		case "standalone":
			wantScreen := *screenFlagShort || *screenFlagLong || hasScreenFlag(args[1:])
			if !*isDaemonWorker {
				spawnBackgroundStandalone(wantScreen)
				return
			}
			runStandaloneDaemonWorker(wantScreen)
			return
		case "snap":
			quiet := hasQuietFlag(args[1:])
			roomID := resolveActiveRoomID(args[1:], *roomFlag)
			relayService := service.NewRelayService("")
			b64Img, err := service.CaptureScreenSilent()
			if err != nil {
				if !quiet {
					fmt.Printf("[Error] Screen capture failed: %v\n", err)
				}
				return
			}
			if err := relayService.UploadScreenshot(roomID, b64Img); err != nil {
				if !quiet {
					fmt.Printf("[Error] Snap upload failed: %v\n", err)
				}
			} else if !quiet {
				fmt.Printf("[SUCCESS] Screenshot uploaded to room '%s' via Relay!\n", roomID)
			}
			return
		case "text":
			quiet := hasQuietFlag(args[1:])
			roomID := resolveActiveRoomID(args[1:], *roomFlag)
			relayService := service.NewRelayService("")
			clipText, err := clipboard.ReadAll()
			if err != nil || clipText == "" {
				if !quiet {
					fmt.Println("[Error] Clipboard is empty or unreadable!")
				}
				return
			}
			if err := relayService.UploadQuestionText(roomID, clipText); err != nil {
				if !quiet {
					fmt.Printf("[Error] Text upload failed: %v\n", err)
				}
			} else if !quiet {
				fmt.Printf("[SUCCESS] Clipboard text uploaded to room '%s' via Relay!\n", roomID)
			}
			return
		case "fetch":
			quiet := hasQuietFlag(args[1:])
			if err := service.TriggerFetchIPC(); err != nil {
				// Fallback if daemon is not running: fetch directly from relay server
				roomID := resolveActiveRoomID(args[1:], *roomFlag)
				relayService := service.NewRelayService("")
				text, err := relayService.FetchLatestWebText(roomID)
				if err == nil && text != "" {
					_ = clipboard.WriteAll(text)
					if !quiet {
						fmt.Printf("[SUCCESS] Fetched Text and updated PC clipboard: \"%s\"\n", text)
					}
					return
				}
				if !quiet {
					fmt.Println("[Error] Could not fetch Text from relay server!")
				}
			} else if !quiet {
				fmt.Println("[SUCCESS] Triggered fetch & updated PC system clipboard with Text!")
			}
			return
		case "logs":
			tail := false
			for _, a := range args[1:] {
				if a == "-t" || a == "--tail" || a == "-f" {
					tail = true
				}
			}
			service.StreamLogs(tail)
			return
		}
	}

	roomID := *roomFlag
	if roomID == "" && len(args) > 0 && args[0] != "config" && args[0] != "setup" && args[0] != "status" && args[0] != "stop" && args[0] != "logs" && args[0] != "snap" && args[0] != "text" && args[0] != "fetch" && args[0] != "standalone" {
		roomID = args[0]
	}

	if roomID == "" {
		printUsage()
		return
	}

	wantScreen := *screenFlagShort || *screenFlagLong

	// If user ran interactively (without --daemon flag), validate connection first before spawning background process!
	if !*isDaemonWorker {
		spawnBackgroundDaemon(roomID, wantScreen)
		return
	}

	// Internal background daemon process worker loop
	runDaemon(roomID, wantScreen)
}

func hasScreenFlag(args []string) bool {
	for _, a := range args {
		if a == "-s" || a == "--screen" {
			return true
		}
	}
	return false
}

func checkServiceAlreadyRunning() {
	if state, err := service.LoadState(); err == nil && state.PID > 0 {
		client := http.Client{Timeout: 1 * time.Second}
		resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%s/status", state.Port))
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			modeName := state.Mode
			if modeName == "" {
				modeName = "firebase"
			}
			fmt.Printf("[Error] A ctrlv service is ALREADY running (PID: %d, Mode: %s)!\n-> Run 'ctrlv stop' first to stop active service before starting another mode.\n", state.PID, modeName)
			os.Exit(1)
		}
	}
}

func spawnBackgroundStandalone(wantScreen bool) {
	checkServiceAlreadyRunning()

	fmt.Println("==================================================")
	fmt.Println("       ctrlv Standalone Direct AI Service         ")
	fmt.Println("==================================================")
	fmt.Printf(" Loading AI Configuration (%s)...\n", service.GetConfigPath())

	cfg, err := service.LoadAIConfig()
	if err != nil {
		fmt.Printf("[Error] %v\n", err)
		fmt.Printf("Please edit your credentials in %s and run 'ctrlv standalone' again.\n", service.GetConfigPath())
		os.Exit(1)
	}

	fmt.Printf(" AI Provider : %s\n", cfg.Provider)
	fmt.Printf(" Model       : %s\n", cfg.Model)
	fmt.Println(" Testing API Ping connection to AI Provider...")

	if err := service.TestAICredentials(cfg); err != nil {
		fmt.Printf("[Error] AI API Ping test failed: %v\n", err)
		fmt.Printf("Please check your API key in %s\n", service.GetConfigPath())
		os.Exit(1)
	}

	fmt.Println("[SUCCESS] AI Credentials Verified! (1-Second API Ping Passed)")

	execPath, err := os.Executable()
	if err != nil {
		log.Fatalf("Failed to resolve executable path: %v", err)
	}

	cmdArgs := []string{"--daemon", "standalone"}
	if wantScreen {
		cmdArgs = append(cmdArgs, "-s")
	}

	cmd := exec.Command(execPath, cmdArgs...)
	setDetachedSysProcAttr(cmd)

	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start background process: %v", err)
	}

	fmt.Println("==================================================")
	fmt.Printf("           ctrlv Standalone Started               \n")
	fmt.Println("==================================================")
	fmt.Printf(" Background Process PID: %d\n", cmd.Process.Pid)
	fmt.Println(" Status                : Active (Running in background)")
	fmt.Println(" Hotkeys Active        : Ctrl + Shift + S (Capture & Direct AI Solve)")
	fmt.Println("                       : Ctrl + Shift + T (Send Clipboard Text & AI Solve)")
	if wantScreen {
		fmt.Println(" Stealth Protection    : ON (Screen-Share Invisible Overlay Active!)")
	}
	fmt.Println(" CLI Triggers          : ctrlv snap | ctrlv text | ctrlv fetch")
	fmt.Println(" Run 'ctrlv status' to view status")
	fmt.Println(" Run 'ctrlv logs -t' to stream live logs in real time")
	fmt.Println(" Run 'ctrlv stop' to stop background service")
	fmt.Println("==================================================")
}

func runStandaloneDaemonWorker(wantScreen bool) {
	log.SetOutput(service.GlobalLogHub)

	cfg, err := service.LoadAIConfig()
	if err != nil {
		log.Fatalf("Error loading AI config: %v", err)
	}

	log.Printf("[Daemon] Standalone AI worker started (PID: %d, Provider: %s, Model: %s)", os.Getpid(), cfg.Provider, cfg.Model)

	if wantScreen {
		go service.LaunchStealthOverlay("Standalone AI")
	}

	stopChan := make(chan struct{})

	// Screenshot solve callback (Ctrl + Shift + S)
	onScreenshot := func() {
		log.Println("[Standalone AI] Capturing screen...")
		if wantScreen {
			service.UpdateOverlayStatus("Capturing Screen...")
		}

		b64Img, err := service.CaptureScreenSilent()
		if err != nil {
			log.Printf("[Capture Error] %v", err)
			if wantScreen {
				service.UpdateOverlayStatus("Capture Error!")
			}
			return
		}

		log.Println("[Standalone AI] Solving screenshot with AI Vision REST API...")
		if wantScreen {
			service.UpdateOverlayStatus("AI Analyzing Screenshot...")
		}

		solution, err := service.SolveScreenshotDirect(cfg, b64Img)
		if err != nil {
			log.Printf("[AI Error] %v", err)
			if wantScreen {
				service.UpdateOverlayStatus("AI Error: " + err.Error())
			}
			return
		}

		if err := clipboard.WriteAll(solution); err != nil {
			log.Printf("[Clipboard Error] %v", err)
		} else {
			log.Println("[Standalone AI] Solution copied directly to PC system clipboard!")
		}

		if wantScreen {
			service.UpdateOverlayText(solution)
			service.UpdateOverlayStatus("AI Solved & Copied to Clipboard!")
		}
	}

	// Clipboard text question solve callback (Ctrl + Shift + T)
	onSendText := func() {
		log.Println("[Standalone AI] Reading PC clipboard question text...")
		if wantScreen {
			service.UpdateOverlayStatus("Reading Clipboard Text...")
		}

		clipText, err := clipboard.ReadAll()
		if err != nil || clipText == "" {
			log.Printf("[Clipboard Error] Failed to read text from clipboard: %v", err)
			if wantScreen {
				service.UpdateOverlayStatus("Clipboard is Empty!")
			}
			return
		}

		log.Printf("[Standalone AI] Solving text question: \"%s\"", clipText)
		if wantScreen {
			service.UpdateOverlayStatus("AI Solving Text Question...")
		}

		solution, err := service.SolveTextDirect(cfg, clipText)
		if err != nil {
			log.Printf("[AI Error] %v", err)
			if wantScreen {
				service.UpdateOverlayStatus("AI Error: " + err.Error())
			}
			return
		}

		if err := clipboard.WriteAll(solution); err != nil {
			log.Printf("[Clipboard Error] %v", err)
		} else {
			log.Println("[Standalone AI] Solution copied directly to PC system clipboard!")
		}

		if wantScreen {
			service.UpdateOverlayText(solution)
			service.UpdateOverlayStatus("AI Solved Text & Copied!")
		}
	}

	ipcServer := service.NewIPCServer("standalone", "standalone", stopChan, onScreenshot, func() {}, onSendText)
	if err := ipcServer.Start(); err != nil {
		log.Printf("IPC Server warning: %v", err)
	}
	defer ipcServer.Stop()

	hotkeyHandler := service.NewHotkeyHandler(onScreenshot, func() {}, onSendText, service.ToggleOverlayVisibility)
	go hotkeyHandler.Start()
	defer hotkeyHandler.Stop()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		log.Printf("[Daemon] Received signal %v. Shutting down...", sig)
	case <-stopChan:
		log.Println("[Daemon] Shutdown request received. Exiting...")
	}
}

func resolveActiveRoomID(subArgs []string, flagRoom string) string {
	if flagRoom != "" {
		return flagRoom
	}
	for _, a := range subArgs {
		if a != "-q" && a != "--quiet" && a != "-silent" {
			return a
		}
	}
	// Fallback to active daemon state file
	if state, err := service.LoadState(); err == nil && state.RoomID != "" {
		return state.RoomID
	}
	return "room-alpha-123"
}

func hasQuietFlag(args []string) bool {
	for _, a := range args {
		if a == "-q" || a == "--quiet" || a == "-silent" {
			return true
		}
	}
	return false
}

func spawnBackgroundDaemon(roomID string, wantScreen bool) {
	checkServiceAlreadyRunning()

	fmt.Println("Verifying WebSocket Relay Server connection...")
	relayService := service.NewRelayService("")

	if err := relayService.VerifyConnection(roomID); err != nil {
		fmt.Printf("[Error] Relay Server connection verification failed: %v\nExiting without starting background process.\n", err)
		relayService.Close()
		os.Exit(1)
	}
	relayService.Close()
	fmt.Println("[Relay] WebSocket Connection verified successfully!")

	execPath, err := os.Executable()
	if err != nil {
		log.Fatalf("Failed to resolve executable path: %v", err)
	}

	cmdArgs := []string{"--daemon", "-r", roomID}
	if wantScreen {
		cmdArgs = append(cmdArgs, "-s")
	}

	cmd := exec.Command(execPath, cmdArgs...)
	setDetachedSysProcAttr(cmd)

	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start background process: %v", err)
	}

	fmt.Println("==================================================")
	fmt.Printf("           ctrlv Service Started (Relay Mode)     \n")
	fmt.Println("==================================================")
	fmt.Printf(" Connecting to Room ID : %s\n", roomID)
	fmt.Printf(" Relay Server          : %s\n", service.GetRelayURL())
	fmt.Printf(" Background Process PID: %d\n", cmd.Process.Pid)
	fmt.Println(" Status                : Active (Running in background)")
	fmt.Println(" Realtime Auto-Push    : Active (Text automatically copied to PC clipboard)")
	fmt.Println(" Hotkeys Active        : Ctrl + Shift + S (Screenshot)")
	fmt.Println("                       : Ctrl + Shift + T (Send Clipboard Text Question)")
	fmt.Println("                       : Ctrl + Shift + F (Re-Copy Clipboard Text)")
	if wantScreen {
		fmt.Println(" Stealth Protection    : ON (Screen-Share Invisible Overlay Active!)")
	}
	fmt.Println(" CLI Triggers          : ctrlv snap | ctrlv text | ctrlv fetch")
	fmt.Println(" Run 'ctrlv status' to view status")
	fmt.Println(" Run 'ctrlv logs -t' to stream live logs in real time")
	fmt.Println(" Run 'ctrlv stop' to stop background service")
	fmt.Println("==================================================")
}

func runDaemon(roomID string, wantScreen bool) {
	// Direct all log output to in-memory broadcast hub (Zero Disk Files)
	log.SetOutput(service.GlobalLogHub)

	log.Printf("[Daemon] Relay worker started for Room ID: %s (PID: %d)", roomID, os.Getpid())

	if wantScreen {
		go service.LaunchStealthOverlay(roomID)
	}

	relayService := service.NewRelayService("")
	defer relayService.Close()

	stopChan := make(chan struct{})

	ctx, cancelListener := context.WithCancel(context.Background())
	defer cancelListener()

	var latestWebText string
	var latestWebTextMu sync.RWMutex

	go relayService.ListenRoomUpdates(ctx, roomID, func(msgType string, rawText string, senderID string) {
		// Ignore PC -> Web ("exe_web") text and messages sent by THIS CLI instance
		if msgType == "exe_web" || (senderID != "" && senderID == relayService.InstanceID) {
			return
		}

		// Process Web -> PC ("web_exe" or fallback "text") incoming payloads
		if msgType == "web_exe" || msgType == "text" {
			cleanText := rawText
			if len(cleanText) > 6 && cleanText[len(cleanText)-6:] == "\n/---/" {
				cleanText = cleanText[:len(cleanText)-6]
			} else if len(cleanText) > 5 && cleanText[len(cleanText)-5:] == "/---/" {
				cleanText = cleanText[:len(cleanText)-5]
			}

			latestWebTextMu.Lock()
			latestWebText = cleanText
			latestWebTextMu.Unlock()

			if err := clipboard.WriteAll(cleanText); err != nil {
				log.Printf("[Realtime Clipboard Error] Failed to write text: %v", err)
			} else {
				log.Printf("[Realtime Auto-Push] Automatically copied text to PC clipboard: \"%s\"", cleanText)
			}

			if wantScreen {
				service.UpdateOverlayText(cleanText)
				service.UpdateOverlayStatus("Text Received & Copied to Clipboard!")
			}
		}
	})

	// Callback for Screenshot (Ctrl + Shift + S)
	onScreenshot := func() {
		if wantScreen {
			service.UpdateOverlayStatus("Capturing & Sending Screenshot...")
		}
		b64Img, err := service.CaptureScreenSilent()
		if err != nil {
			log.Printf("[Capture Error] %v", err)
			if wantScreen {
				service.UpdateOverlayStatus("Capture Error!")
			}
			return
		}
		if err := relayService.UploadScreenshot(roomID, b64Img); err != nil {
			log.Printf("[Upload Error] %v", err)
			if wantScreen {
				service.UpdateOverlayStatus("Upload Error!")
			}
		} else {
			if wantScreen {
				service.UpdateOverlayStatus("Screenshot Sent to Web!")
			}
		}
	}

	// Callback for Manual Re-Fetch Text (Ctrl + Shift + F)
	onFetchText := func() {
		latestWebTextMu.RLock()
		text := latestWebText
		latestWebTextMu.RUnlock()

		if text == "" {
			if wantScreen {
				service.UpdateOverlayStatus("Querying Relay for Web Text...")
			}
			fetched, err := relayService.FetchLatestWebText(roomID)
			if err == nil && fetched != "" {
				text = fetched
				latestWebTextMu.Lock()
				latestWebText = fetched
				latestWebTextMu.Unlock()
			}
		}

		if text == "" {
			if wantScreen {
				service.UpdateOverlayStatus("No Web Text Available Yet!")
			}
			return
		}

		if err := clipboard.WriteAll(text); err != nil {
			log.Printf("[Clipboard Write Error] %v", err)
		} else {
			log.Printf("[Fetch Text] Copied web text to PC clipboard: \"%s\"", text)
		}

		if wantScreen {
			service.UpdateOverlayText(text)
			service.UpdateOverlayStatus("Fetched Web Text & Copied!")
		}
	}

	// Callback for Send Clipboard Text Question (Ctrl + Shift + T)
	onSendText := func() {
		log.Println("[Daemon] Reading PC clipboard text...")
		if wantScreen {
			service.UpdateOverlayStatus("Reading PC Clipboard Text...")
		}

		clipText, err := clipboard.ReadAll()
		if err != nil || clipText == "" {
			log.Printf("[Clipboard Error] Empty clipboard: %v", err)
			if wantScreen {
				service.UpdateOverlayStatus("Clipboard Empty!")
			}
			return
		}

		if err := relayService.UploadQuestionText(roomID, clipText); err != nil {
			log.Printf("[Upload Error] Failed to upload text: %v", err)
			if wantScreen {
				service.UpdateOverlayStatus("Failed to Upload Text!")
			}
		} else {
			log.Printf("[Relay] Uploaded PC clipboard text: \"%s\"", clipText)
			if wantScreen {
				service.UpdateOverlayStatus("Sent Text to Web!")
			}
		}
	}

	ipcServer := service.NewIPCServer("relay", roomID, stopChan, onScreenshot, onFetchText, onSendText)
	if err := ipcServer.Start(); err != nil {
		log.Printf("IPC Server warning: %v", err)
	}
	defer ipcServer.Stop()

	hotkeyHandler := service.NewHotkeyHandler(onScreenshot, onFetchText, onSendText, service.ToggleOverlayVisibility)
	go hotkeyHandler.Start()
	defer hotkeyHandler.Stop()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		log.Printf("[Daemon] Received signal %v. Shutting down...", sig)
	case <-stopChan:
		log.Println("[Daemon] Shutdown request received. Exiting...")
	}
}

func printUsage() {
	fmt.Println("ctrlv - Realtime Cross-Device Screenshot & Clipboard Sync CLI")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  ctrlv config             Open ctrlv_config.json in text editor (notepad/nano/saved editor)")
	fmt.Println("  ctrlv config -e <editor>  Set preferred editor (e.g. code, notepad, nano) & open config")
	fmt.Println("  ctrlv standalone        Run Direct AI mode (Direct AI solve)")
	fmt.Println("  ctrlv standalone -s     Run Direct AI mode + Screen-Share Invisible Overlay Notepad")
	fmt.Println("  ctrlv -r <roomid>       Start Room sync background service (Zero-Config Relay)")
	fmt.Println("  ctrlv -r <roomid> -s    Start Room sync + Stealth Overlay Notepad")
	fmt.Println("  ctrlv setup             Auto-configure GNOME silent shortcuts & disable camera sound (Linux)")
	fmt.Println("  ctrlv status            Check if ctrlv service is currently running")
	fmt.Println("  ctrlv snap              Trigger silent screen capture & upload to room")
	fmt.Println("  ctrlv text              Trigger sending clipboard question text to room")
	fmt.Println("  ctrlv fetch             Re-read current text from system clipboard")
	fmt.Println("  ctrlv logs              View current in-memory daemon logs")
	fmt.Println("  ctrlv logs -t           Stream live in-memory logs in real time")
	fmt.Println("  ctrlv stop              Stop the running ctrlv background service")
	fmt.Println("  ctrlv -v, --version     Print version information")
	fmt.Println("  ctrlv -h, --help        Show this help screen")
	fmt.Println()
	fmt.Println("Hotkeys when running:")
	fmt.Println("  Ctrl + Shift + S        Silently capture screen & solve/upload")
	fmt.Println("  Ctrl + Shift + T        Send PC clipboard text question to room")
	fmt.Println("  Ctrl + Shift + F        Re-sync current text on clipboard")
}
