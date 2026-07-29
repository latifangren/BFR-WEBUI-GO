package terminal

import (
	"log"
	"net/http"
	"os/exec"
	"sync"

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

	cmd := exec.Command("su")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		cmd = exec.Command("/system/bin/sh")
		stdin, err = cmd.StdinPipe()
		if err != nil {
			_ = conn.WriteMessage(websocket.TextMessage, []byte("Failed to initialize shell stdin\r\n"))
			return
		}
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("Failed to initialize shell stdout\r\n"))
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("Failed to initialize shell stderr\r\n"))
		return
	}

	if err := cmd.Start(); err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("Failed to start shell process\r\n"))
		return
	}

	var writeMux sync.Mutex
	writeWS := func(data []byte) {
		writeMux.Lock()
		defer writeMux.Unlock()
		_ = conn.WriteMessage(websocket.TextMessage, data)
	}

	// Read stdout
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				writeWS(buf[:n])
			}
			if err != nil {
				break
			}
		}
	}()

	// Read stderr
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stderr.Read(buf)
			if n > 0 {
				writeWS(buf[:n])
			}
			if err != nil {
				break
			}
		}
	}()

	// Read WS messages and write to stdin
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}
		_, _ = stdin.Write(msg)
	}

	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}
