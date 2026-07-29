package terminal

import (
	"log"
	"net/http"
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

	cmd := exec.Command("su")
	f, err := pty.Start(cmd)
	if err != nil {
		cmd = exec.Command("/system/bin/sh")
		f, err = pty.Start(cmd)
		if err != nil {
			_ = conn.WriteMessage(websocket.TextMessage, []byte("Failed to initialize shell interactive PTY\r\n"))
			return
		}
	}
	defer func() {
		_ = f.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}()

	var writeMux sync.Mutex
	writeWS := func(data []byte) error {
		writeMux.Lock()
		defer writeMux.Unlock()
		return conn.WriteMessage(websocket.TextMessage, data)
	}

	// Read from PTY master file and send to Websocket
	go func() {
		buf := make([]byte, 2048)
		for {
			n, err := f.Read(buf)
			if n > 0 {
				if err := writeWS(buf[:n]); err != nil {
					return
				}
			}
			if err != nil {
				// EOF or exit closed
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
		if _, err := f.Write(msg); err != nil {
			break
		}
	}
}

// Keep it backward compatible if any files depend on old exports
func HandleWebsocketStdinOnly(w http.ResponseWriter, r *http.Request) {
	HandleWebsocket(w, r)
}
