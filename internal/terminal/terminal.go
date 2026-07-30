package terminal

import (
	"log"
	"net/http"
	"os"
	"os/exec"
	"sync"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func HandleWebsocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Terminal WS upgrade error: %v", err)
		return
	}
	defer conn.Close()

	var writeMux sync.Mutex
	writeWS := func(data []byte) error {
		writeMux.Lock()
		defer writeMux.Unlock()
		return conn.WriteMessage(websocket.TextMessage, data)
	}

	var cmd *exec.Cmd
	var shellFile *os.File
	var usePty bool

	// 1. Array of shells to try with PTY
	shellCommandFunc := func() (*exec.Cmd, bool) {
		// Try su first
		cmdSu := exec.Command("su")
		if f, err := pty.Start(cmdSu); err == nil {
			shellFile = f
			cmd = cmdSu
			return cmdSu, true
		}

		// Try Busybox sh locations
		busyboxPaths := []string{
			"/data/adb/magisk/busybox",
			"/data/adb/apatch/busybox",
			"/data/adb/ksu/bin/busybox",
			"/system/bin/busybox",
			"/system/xbin/busybox",
		}
		for _, bPath := range busyboxPaths {
			if _, err := os.Stat(bPath); err == nil {
				cmdBb := exec.Command(bPath, "sh")
				if f, err := pty.Start(cmdBb); err == nil {
					shellFile = f
					cmd = cmdBb
					return cmdBb, true
				}
			}
		}

		// Try default system/bin/sh
		cmdSh := exec.Command("/system/bin/sh")
		if f, err := pty.Start(cmdSh); err == nil {
			shellFile = f
			cmd = cmdSh
			return cmdSh, true
		}

		return nil, false
	}

	cmd, usePty = shellCommandFunc()

	if usePty && shellFile != nil {
		// PTY succeeded initialization
		defer func() {
			_ = shellFile.Close()
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
		}()

		// Read from PTY master file and send to Websocket
		go func() {
			buf := make([]byte, 2048)
			for {
				n, err := shellFile.Read(buf)
				if n > 0 {
					if err := writeWS(buf[:n]); err != nil {
						return
					}
				}
				if err != nil {
					break
				}
			}
		}()

		// Read from WebSocket and write to PTY master file
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				break
			}
			if _, err := shellFile.Write(msg); err != nil {
				break
			}
		}
	} else {
		// 2. SELinux blocked /dev/ptmx or /dev/pts. Fallback to standard bi-directional Pipes (SELinux Immune)
		cmd = exec.Command("su")
		stdinPipe, err := cmd.StdinPipe()
		if err != nil {
			cmd = exec.Command("/system/bin/sh", "-i")
			stdinPipe, err = cmd.StdinPipe()
			if err != nil {
				_ = conn.WriteMessage(websocket.TextMessage, []byte("SELinux Blocked all shell allocations\r\n"))
				return
			}
		}

		stdoutPipe, err := cmd.StdoutPipe()
		if err != nil {
			_ = conn.WriteMessage(websocket.TextMessage, []byte("Failed to initialize stdout pipe\r\n"))
			return
		}

		stderrPipe, err := cmd.StderrPipe()
		if err != nil {
			_ = conn.WriteMessage(websocket.TextMessage, []byte("Failed to initialize stderr pipe\r\n"))
			return
		}

		if err := cmd.Start(); err != nil {
			_ = conn.WriteMessage(websocket.TextMessage, []byte("Failed to start fallback root process\r\n"))
			return
		}

		defer func() {
			_ = stdinPipe.Close()
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
		}()

		// Write initial message explaining fallback run
		_ = writeWS([]byte("SELinux restricts TTY dev/ptmx. Swapped raw pipe wrapper session.\r\n"))

		// Forward stdout to Websocket
		go func() {
			buf := make([]byte, 1024)
			for {
				n, err := stdoutPipe.Read(buf)
				if n > 0 {
					_ = writeWS(buf[:n])
				}
				if err != nil {
					break
				}
			}
		}()

		// Forward stderr to Websocket
		go func() {
			buf := make([]byte, 1024)
			for {
				n, err := stderrPipe.Read(buf)
				if n > 0 {
					_ = writeWS(buf[:n])
				}
				if err != nil {
					break
				}
			}
		}()

		// Read Websocket and write to stdin
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				break
			}
			_, err = stdinPipe.Write(msg)
			if err != nil {
				break
			}
		}
	}
}

// Keep it backward compatible if any files depend on old exports
func HandleWebsocketStdinOnly(w http.ResponseWriter, r *http.Request) {
	HandleWebsocket(w, r)
}
