package terminal

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"bfr-webui-go/internal/auth"
	"bfr-webui-go/internal/bufferpool"
	"bfr-webui-go/internal/config"
	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		u, err := url.Parse(origin)
		if err != nil {
			return false
		}
		return strings.EqualFold(u.Host, r.Host)
	},
}

func HandleWebsocket(w http.ResponseWriter, r *http.Request) {
	// Verify session authentication token before upgrading WS
	token := ""
	if cookie, err := r.Cookie(auth.CookieName); err == nil && cookie.Value != "" {
		token = cookie.Value
	}
	if token == "" {
		token = r.URL.Query().Get("token")
	}
	if token == "" || !auth.GetManager().ValidateSession(token) {
		http.Error(w, "Unauthorized session", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Terminal WS upgrade error: %v", err)
		return
	}

	const (
		writeWait  = 10 * time.Second
		pongWait   = 60 * time.Second
		pingPeriod = (pongWait * 9) / 10
	)

	conn.SetReadLimit(512 * 1024)
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	var writeMux sync.Mutex
	writeWS := func(data []byte) error {
		writeMux.Lock()
		defer writeMux.Unlock()
		_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
		return conn.WriteMessage(websocket.TextMessage, data)
	}

	stopPing := make(chan struct{})
	go func() {
		ticker := time.NewTicker(pingPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				writeMux.Lock()
				_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
				err := conn.WriteMessage(websocket.PingMessage, nil)
				writeMux.Unlock()
				if err != nil {
					conn.Close()
					return
				}
			case <-stopPing:
				return
			}
		}
	}()

	var cmd *exec.Cmd
	var shellFile *os.File

	defer func() {
		close(stopPing)
		conn.Close()
		if shellFile != nil {
			_ = shellFile.Close()
		}
		if cmd != nil && cmd.Process != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	}()
	var usePty bool

	// 1. Array of shells to try with PTY
	shellCommandFunc := func() (*exec.Cmd, bool) {
		// Try su first
		cmdSu := exec.Command(config.SUBin)
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
			buf := bufferpool.GetBytes(2048)
			defer bufferpool.PutBytes(buf)
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
		cmd = exec.Command(config.SUBin)
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
			buf := bufferpool.GetBytes(1024)
			defer bufferpool.PutBytes(buf)
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
			buf := bufferpool.GetBytes(1024)
			defer bufferpool.PutBytes(buf)
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
