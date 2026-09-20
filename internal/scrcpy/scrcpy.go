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
	"strconv"
	"strings"
	"sync"
	"time"

	"bfr-webui-go/internal/bufferpool"
	"bfr-webui-go/internal/config"
	"github.com/gorilla/websocket"
)

type InputEvent struct {
	Action    string `json:"action"`
	X         int    `json:"x"`
	Y         int    `json:"y"`
	X2        int    `json:"x2"`
	Y2        int    `json:"y2"`
	Duration  int    `json:"duration"`
	Text      string `json:"text"`
	KeyCode   int    `json:"keycode"`
	PointerID int    `json:"pointerId"`
	On        bool   `json:"on"`
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

func handleHubInput(hub *Hub, evt InputEvent) {
	switch evt.Action {
	case "down":
		_ = hub.SendTouch(0, evt.X, evt.Y, 0)
	case "move":
		_ = hub.SendTouch(2, evt.X, evt.Y, 0)
	case "up":
		_ = hub.SendTouch(1, evt.X, evt.Y, 0)
	case "tap", "click":
		hub.InjectTouch("tap", evt.X, evt.Y, 0, 0, 0)
	case "swipe":
		hub.InjectTouch("swipe", evt.X, evt.Y, evt.X2, evt.Y2, evt.Duration)
	case "key", "keycode":
		hub.InjectKey("", evt.KeyCode)
	case "back", "home", "recents", "app_switch", "power", "vol_up", "vol_down", "enter", "tab", "delete", "backspace", "wake", "sleep":
		hub.InjectKey(evt.Action, 0)
	case "screen_power":
		_ = hub.SetScreenPower(evt.On)
	case "screen_power_off":
		_ = hub.SetScreenPower(false)
	case "screen_power_on":
		_ = hub.SetScreenPower(true)
	case "text":
		safeText := SanitizeInput(evt.Text)
		if safeText != "" {
			_ = exec.Command(config.SUBin, "-c", fmt.Sprintf("input text '%s'", safeText)).Run()
		}
	}
}

func parseQueryConfig(r *http.Request) ScrcpyConfig {
	cfg := DefaultConfig()
	q := r.URL.Query()

	if q.Has("max_size") {
		if v, err := strconv.Atoi(q.Get("max_size")); err == nil && v >= 0 {
			cfg.MaxSize = v
		}
	}
	if q.Has("bit_rate") {
		if v, err := strconv.Atoi(q.Get("bit_rate")); err == nil && v > 0 {
			cfg.BitRate = v
		}
	}
	if v, err := strconv.Atoi(q.Get("max_fps")); err == nil && v > 0 {
		cfg.MaxFps = v
	}
	if v, err := strconv.Atoi(q.Get("display_id")); err == nil && v >= 0 {
		cfg.DisplayID = v
	}
	if q.Get("stay_awake") == "false" {
		cfg.StayAwake = false
	}
	if q.Get("screen_off") == "true" {
		cfg.TurnScreenOff = true
	}
	return cfg
}

func HandleWebsocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Scrcpy WS upgrade error: %v", err)
		return
	}
	defer conn.Close()

	const pongWait = 60 * time.Second
	conn.SetReadLimit(512 * 1024)
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	cfg := parseQueryConfig(r)
	hub := GetHub()
	hubErr := hub.Start(cfg)

	if hubErr == nil {
		client, width, height, videoWidth, videoHeight, lastConfig := hub.RegisterClient(conn)
		defer func() {
			hub.UnregisterClient(client)
			client.Close()
		}()

		// Send stream metadata (WebCodecs H.264 mode)
		meta, _ := json.Marshal(map[string]interface{}{
			"type":        "metadata",
			"width":       width,
			"height":      height,
			"videoWidth":  videoWidth,
			"videoHeight": videoHeight,
			"fps":         cfg.MaxFps,
			"engine":      "webcodecs",
			"codec":       "avc1.640033",
			"max_size":    cfg.MaxSize,
			"bit_rate":    cfg.BitRate,
			"displayId":   cfg.DisplayID,
			"encoder":     GetEncoderInfo(),
		})
		client.Send(websocket.TextMessage, meta)

		// Send cached SPS/PPS packet immediately to warm up browser decoder
		if len(lastConfig) > 0 {
			client.Send(websocket.BinaryMessage, lastConfig)
		}

		// Input event loop
		for {
			_, msgBytes, err := conn.ReadMessage()
			if err != nil {
				break
			}
			var evt InputEvent
			if err := json.Unmarshal(msgBytes, &evt); err == nil {
				handleHubInput(hub, evt)
			}
		}
		return
	}

	// Fallback to legacy screencap stream if scrcpy-server is unavailable
	log.Printf("[Scrcpy] Hub start failed (%v), falling back to legacy screencap stream", hubErr)
	fallbackMeta, _ := json.Marshal(map[string]interface{}{
		"type":   "metadata",
		"width":  1080,
		"height": 2400,
		"fps":    3,
		"engine": "mjpeg",
	})
	_ = conn.WriteMessage(websocket.TextMessage, fallbackMeta)

	done := make(chan struct{})
	var writeMux sync.Mutex

	go func() {
		defer close(done)
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[Scrcpy] Recovered panic in fallback input reader: %v", r)
			}
		}()
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

	defer func() {
		if r := recover(); r != nil {
			log.Printf("[Scrcpy] Recovered panic in fallback screencap loop: %v", r)
		}
	}()

	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			writeMux.Lock()
			frameData, err := captureScreenFrame()
			if err == nil && len(frameData) > 0 {
				_ = conn.SetWriteDeadline(time.Now().Add(1 * time.Second))
				err = conn.WriteMessage(websocket.BinaryMessage, frameData)
			}
			writeMux.Unlock()

			if err != nil {
				return
			}
		}
	}
}
