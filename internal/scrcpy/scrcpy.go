package scrcpy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/jpeg"
	"image/png"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"bfr-webui-go/internal/bufferpool"
	"bfr-webui-go/internal/config"
	"github.com/gorilla/websocket"
)

type InputEvent struct {
	Action   string `json:"action"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
	X2       int    `json:"x2"`
	Y2       int    `json:"y2"`
	Duration int    `json:"duration"`
	Text     string `json:"text"`
	KeyCode  int    `json:"keycode"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func captureScreenFrame() ([]byte, error) {
	out, err := exec.Command(config.SUBin, "-c", "screencap -p").Output()
	if err != nil || len(out) == 0 {
		out, err = exec.Command("/system/bin/screencap", "-p").Output()
		if err != nil || len(out) == 0 {
			return nil, fmt.Errorf("screencap failed: %v", err)
		}
	}

	img, err := png.Decode(bytes.NewReader(out))
	if err == nil {
		buf := bufferpool.GetBuffer()
		defer bufferpool.PutBuffer(buf)
		if err := jpeg.Encode(buf, img, &jpeg.Options{Quality: 50}); err == nil {
			encoded := make([]byte, buf.Len())
			copy(encoded, buf.Bytes())
			return encoded, nil
		}
	}

	return out, nil
}

// M-13: SanitizeInput cleans shell special characters like backtick, dollar sign,
// semicolon, backslash, quotes, and other shell operators from text input.
func SanitizeInput(text string) string {
	if text == "" {
		return ""
	}
	s := strings.ReplaceAll(text, " ", "%s")
	replacer := strings.NewReplacer(
		"`", "",
		"$", "",
		";", "",
		"\\", "",
		"'", "",
		"\"", "",
		"&", "",
		"|", "",
		"<", "",
		">", "",
		"(", "",
		")", "",
		"{", "",
		"}", "",
		"\n", "",
		"\r", "",
	)
	return replacer.Replace(s)
}

func handleInputEvent(evt InputEvent) {
	switch evt.Action {
	case "click", "tap":
		if evt.X >= 0 && evt.Y >= 0 {
			_ = exec.Command(config.SUBin, "-c", fmt.Sprintf("input tap %d %d", evt.X, evt.Y)).Run()
		}
	case "swipe":
		dur := evt.Duration
		if dur <= 0 {
			dur = 300
		}
		_ = exec.Command(config.SUBin, "-c", fmt.Sprintf("input swipe %d %d %d %d %d", evt.X, evt.Y, evt.X2, evt.Y2, dur)).Run()
	case "key", "keycode":
		if evt.KeyCode > 0 {
			_ = exec.Command(config.SUBin, "-c", fmt.Sprintf("input keyevent %d", evt.KeyCode)).Run()
		}
	case "text":
		safeText := SanitizeInput(evt.Text)
		if safeText != "" {
			_ = exec.Command(config.SUBin, "-c", fmt.Sprintf("input text '%s'", safeText)).Run()
		}
	case "back":
		_ = exec.Command(config.SUBin, "-c", "input keyevent 4").Run()
	case "home":
		_ = exec.Command(config.SUBin, "-c", "input keyevent 3").Run()
	case "recents", "app_switch":
		_ = exec.Command(config.SUBin, "-c", "input keyevent 187").Run()
	case "vol_up":
		_ = exec.Command(config.SUBin, "-c", "input keyevent 24").Run()
	case "vol_down":
		_ = exec.Command(config.SUBin, "-c", "input keyevent 25").Run()
	}
}

func HandleWebsocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Scrcpy WS upgrade error: %v", err)
		return
	}
	defer conn.Close()

	done := make(chan struct{})
	var writeMux sync.Mutex

	// Incoming input event reader loop
	go func() {
		defer close(done)
		for {
			_, msgBytes, err := conn.ReadMessage()
			if err != nil {
				break
			}
			var evt InputEvent
			if err := json.Unmarshal(msgBytes, &evt); err == nil {
				go handleInputEvent(evt)
			}
		}
	}()

	// Screen frame capture & stream loop (~3.3 FPS)
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			// Adaptive Frame Throttling: TryLock to skip frame capture/send if previous write is still lagging
			if !writeMux.TryLock() {
				continue
			}

			frameData, err := captureScreenFrame()
			if err == nil && len(frameData) > 0 {
				err = conn.WriteMessage(websocket.BinaryMessage, frameData)
			}
			writeMux.Unlock()

			if err != nil {
				return
			}
		}
	}
}
