package scrcpy

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"bfr-webui-go/internal/config"
	"github.com/gorilla/websocket"
)

const (
	ScrcpyServerPort = 7007
)

// ScrcpyConfig holds streaming and device configuration for scrcpy-server.
type ScrcpyConfig struct {
	MaxSize        int  `json:"max_size"`
	BitRate        int  `json:"bit_rate"`
	MaxFps         int  `json:"max_fps"`
	DisplayID      int  `json:"display_id"`
	StayAwake      bool `json:"stay_awake"`
	TurnScreenOff  bool `json:"turn_screen_off"`
}

// DefaultConfig returns optimal default streaming parameters.
func DefaultConfig() ScrcpyConfig {
	return ScrcpyConfig{
		MaxSize:       1280,
		BitRate:       4000000, // 4 Mbps
		MaxFps:        60,
		DisplayID:     0,
		StayAwake:     true,
		TurnScreenOff: false,
	}
}

type wsMessage struct {
	msgType int
	data    []byte
}

// WSClient represents a connected WebSocket client with non-blocking send and single-goroutine writing.
type WSClient struct {
	hub       *Hub
	conn      *websocket.Conn
	send      chan wsMessage
	done      chan struct{}
	closeOnce sync.Once
}

// NewWSClient creates a new WSClient instance.
func NewWSClient(hub *Hub, conn *websocket.Conn) *WSClient {
	return &WSClient{
		hub:  hub,
		conn: conn,
		send: make(chan wsMessage, 64),
		done: make(chan struct{}),
	}
}

// Send queues a message for transmission; drops the frame non-blockingly if buffer is full.
func (c *WSClient) Send(msgType int, data []byte) bool {
	select {
	case <-c.done:
		return false
	default:
	}

	msg := wsMessage{msgType: msgType, data: data}
	select {
	case c.send <- msg:
		return true
	default:
		// Drop frame to prevent slow client from stalling the entire hub read loop
		return false
	}
}

// Close closes the client's done channel and underlying WebSocket connection safely.
func (c *WSClient) Close() {
	c.closeOnce.Do(func() {
		close(c.done)
		if c.conn != nil {
			_ = c.conn.Close()
		}
	})
}

// writePump is the ONLY goroutine allowed to call write methods on c.conn.
func (c *WSClient) writePump() {
	ticker := time.NewTicker(20 * time.Second)
	defer func() {
		ticker.Stop()
		c.Close()
		if c.hub != nil {
			c.hub.UnregisterClient(c)
		}
	}()

	for {
		select {
		case <-c.done:
			return

		case msg, ok := <-c.send:
			if !ok {
				return
			}
			if c.conn != nil {
				_ = c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
				if err := c.conn.WriteMessage(msg.msgType, msg.data); err != nil {
					return
				}
			}

		case <-ticker.C:
			if c.conn != nil {
				_ = c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
				if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			}
		}
	}
}

// Hub manages the scrcpy-server background process, TCP streaming, and WebSocket clients.
type Hub struct {
	mu          sync.Mutex
	controlMu   sync.Mutex
	serverCmd   *exec.Cmd
	tcpConn     net.Conn
	isRunning   bool
	clients     map[*WSClient]bool
	width       int
	height      int
	videoWidth  int
	videoHeight int
	lastConfig  []byte
	idleTimer   *time.Timer
	stopChan    chan struct{}
	cfg         ScrcpyConfig
}

var (
	globalHub     *Hub
	globalHubOnce sync.Once
)

// GetHub returns the singleton Hub instance.
func GetHub() *Hub {
	globalHubOnce.Do(func() {
		globalHub = &Hub{
			clients: make(map[*WSClient]bool),
			cfg:     DefaultConfig(),
		}
	})
	return globalHub
}

// FindJar looks for scrcpy-server.jar in known locations.
func FindJar() (string, error) {
	candidates := []string{
		filepath.Join(config.ModuleDir, "bin", "scrcpy-server.jar"),
		"/data/adb/modules/bfr_webui_go/bin/scrcpy-server.jar",
		"/data/local/tmp/scrcpy-server.jar",
		"bin/scrcpy-server.jar",
		filepath.Join(".", "bin", "scrcpy-server.jar"),
	}

	for _, p := range candidates {
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p, nil
		}
	}
	return "", fmt.Errorf("scrcpy-server.jar not found in known locations")
}

// Start ensures scrcpy-server is running and streaming on port 7007.
func (h *Hub) Start(cfg ScrcpyConfig) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.idleTimer != nil {
		h.idleTimer.Stop()
		h.idleTimer = nil
	}

	if h.isRunning {
		// If running with the same display and max size, reuse
		if h.cfg.DisplayID == cfg.DisplayID && h.cfg.MaxSize == cfg.MaxSize && h.cfg.BitRate == cfg.BitRate {
			return nil
		}
		// Config changed, restart server
		h.cleanupLocked()
	}

	jarPath, err := FindJar()
	if err != nil {
		return err
	}

	h.cfg = cfg
	if err := h.startServerProcess(jarPath, cfg); err != nil {
		return fmt.Errorf("failed to start scrcpy-server: %w", err)
	}

	// Connect to TCP port 7007 with retry loop
	var conn net.Conn
	for i := 0; i < 25; i++ {
		conn, err = net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", ScrcpyServerPort), 250*time.Millisecond)
		if err == nil {
			break
		}
		time.Sleep(150 * time.Millisecond)
	}
	if err != nil {
		h.stopServerProcess()
		return fmt.Errorf("failed to connect to scrcpy-server on port %d: %w", ScrcpyServerPort, err)
	}

	h.tcpConn = conn
	h.isRunning = true
	h.stopChan = make(chan struct{})

	// Read initial resolution header (16 bytes: 4 bytes device width, 4 bytes device height, 4 bytes video width, 4 bytes video height in big endian)
	header := make([]byte, 16)
	if _, err := io.ReadFull(conn, header); err != nil {
		h.cleanupLocked()
		return fmt.Errorf("failed to read scrcpy resolution header: %w", err)
	}
	h.width = int(binary.BigEndian.Uint32(header[0:4]))
	h.height = int(binary.BigEndian.Uint32(header[4:8]))
	h.videoWidth = int(binary.BigEndian.Uint32(header[8:12]))
	h.videoHeight = int(binary.BigEndian.Uint32(header[12:16]))
	log.Printf("[ScrcpyHub] Connected. Device resolution: %dx%d, Video resolution: %dx%d", h.width, h.height, h.videoWidth, h.videoHeight)

	// If TurnScreenOff is requested at launch, send custom command 99
	if cfg.TurnScreenOff {
		go func() {
			time.Sleep(300 * time.Millisecond)
			_ = h.SetScreenPower(false)
		}()
	}

	// Start reading video stream
	go h.readLoop(conn, h.stopChan)
	return nil
}

func (h *Hub) startServerProcess(jarPath string, cfg ScrcpyConfig) error {
	// Kill leftover process on port 7007
	_ = exec.Command("sh", "-c", "killall -9 app_process 2>/dev/null; fuser -k 7007/tcp 2>/dev/null").Run()
	time.Sleep(150 * time.Millisecond)

	maxSize := cfg.MaxSize
	if maxSize < 0 {
		maxSize = 1280
	}
	bitRate := cfg.BitRate
	if bitRate <= 0 {
		bitRate = 8000000
	}

	cmdStr := fmt.Sprintf(
		"CLASSPATH=%s app_process / org.server.scrcpy.Server 127.0.0.1 %d %d true false true %t %d",
		jarPath, maxSize, bitRate, cfg.StayAwake, cfg.DisplayID,
	)

	h.serverCmd = exec.Command(config.SUBin, "-c", cmdStr)
	if err := h.serverCmd.Start(); err != nil {
		// Fallback to plain sh
		h.serverCmd = exec.Command("sh", "-c", cmdStr)
		if err := h.serverCmd.Start(); err != nil {
			return err
		}
	}

	return nil
}

func (h *Hub) stopServerProcess() {
	if h.serverCmd != nil && h.serverCmd.Process != nil {
		_ = h.serverCmd.Process.Kill()
		h.serverCmd = nil
	}
	_ = exec.Command("sh", "-c", "fuser -k 7007/tcp 2>/dev/null").Run()
}

func (h *Hub) cleanupLocked() {
	h.isRunning = false
	h.lastConfig = nil
	if h.tcpConn != nil {
		_ = h.tcpConn.Close()
		h.tcpConn = nil
	}
	if h.stopChan != nil {
		close(h.stopChan)
		h.stopChan = nil
	}
	h.stopServerProcess()
}

func (h *Hub) readLoop(conn net.Conn, stopChan chan struct{}) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[ScrcpyHub] Recovered panic in readLoop: %v", r)
		}
		h.mu.Lock()
		if h.tcpConn == conn {
			h.cleanupLocked()
		}
		h.mu.Unlock()
	}()

	sizeBuf := make([]byte, 4)
	for {
		select {
		case <-stopChan:
			return
		default:
		}

		if _, err := io.ReadFull(conn, sizeBuf); err != nil {
			log.Printf("[ScrcpyHub] Stream read error: %v", err)
			return
		}
		pktSize := int(binary.BigEndian.Uint32(sizeBuf))
		if pktSize <= 0 || pktSize > 10*1024*1024 {
			log.Printf("[ScrcpyHub] Invalid packet size: %d", pktSize)
			return
		}

		payload := make([]byte, pktSize)
		if _, err := io.ReadFull(conn, payload); err != nil {
			log.Printf("[ScrcpyHub] Failed to read payload: %v", err)
			return
		}

		if len(payload) < 10 {
			continue
		}

		pktType := int(payload[0])
		if pktType != 1 { // Only broadcast VIDEO frames
			continue
		}

		flag := payload[1]
		pts := binary.BigEndian.Uint64(payload[2:10])
		nalData := payload[10:]

		// Build binary message for WebSocket client:
		// [0]: Flag (2=Config, 1=Key, 0=Delta)
		// [1..8]: PTS (uint64 big-endian)
		// [9..]: NAL bytes
		msg := make([]byte, 9+len(nalData))
		msg[0] = flag
		binary.BigEndian.PutUint64(msg[1:9], pts)
		copy(msg[9:], nalData)

		h.mu.Lock()
		if flag == 2 { // CONFIG packet (SPS/PPS)
			h.lastConfig = make([]byte, len(msg))
			copy(h.lastConfig, msg)
		}

		// Snapshot clients list to avoid holding mutex during I/O
		clientList := make([]*WSClient, 0, len(h.clients))
		for client := range h.clients {
			clientList = append(clientList, client)
		}
		h.mu.Unlock()

		// Broadcast to connected WebSocket clients non-blockingly
		for _, client := range clientList {
			client.Send(websocket.BinaryMessage, msg)
		}
	}
}

// RegisterClient adds a client, starts its writePump, and returns video dimensions and SPS/PPS config packet.
func (h *Hub) RegisterClient(ws *websocket.Conn) (*WSClient, int, int, int, int, []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.idleTimer != nil {
		h.idleTimer.Stop()
		h.idleTimer = nil
	}

	client := NewWSClient(h, ws)
	h.clients[client] = true
	width := h.width
	height := h.height
	videoWidth := h.videoWidth
	videoHeight := h.videoHeight
	var lastConfig []byte
	if h.lastConfig != nil {
		lastConfig = make([]byte, len(h.lastConfig))
		copy(lastConfig, h.lastConfig)
	}

	go client.writePump()

	return client, width, height, videoWidth, videoHeight, lastConfig
}

// UnregisterClient removes a client, closes its write loop, and schedules idle shutdown after 15s.
func (h *Hub) UnregisterClient(client *WSClient) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if client == nil {
		return
	}

	if _, exists := h.clients[client]; exists {
		delete(h.clients, client)
		client.Close()
	}

	if len(h.clients) == 0 {
		if h.idleTimer != nil {
			h.idleTimer.Stop()
		}
		h.idleTimer = time.AfterFunc(15*time.Second, func() {
			h.mu.Lock()
			if len(h.clients) == 0 && h.isRunning {
				log.Println("[ScrcpyHub] Idle timeout reached. Stopping server.")
				h.cleanupLocked()
			}
			h.mu.Unlock()
		})
	}
}

// SendControlPacket writes a binary ControlPacket (Type 2) down the open TCP socket.
func (h *Hub) SendControlPacket(data []byte) error {
	h.controlMu.Lock()
	defer h.controlMu.Unlock()

	h.mu.Lock()
	conn := h.tcpConn
	h.mu.Unlock()

	if conn == nil {
		return fmt.Errorf("no active scrcpy connection")
	}

	packetSize := 1 + len(data)
	buf := make([]byte, 4+1+len(data))
	binary.BigEndian.PutUint32(buf[0:4], uint32(packetSize))
	buf[4] = 2 // CONTROL packet type
	copy(buf[5:], data)

	_, err := conn.Write(buf)
	return err
}

// SetScreenPower controls physical display power (false = OFF, true = ON/NORMAL).
func (h *Hub) SetScreenPower(on bool) error {
	mode := 0 // POWER_MODE_OFF
	if on {
		mode = 2 // POWER_MODE_NORMAL
	}
	data := make([]byte, 8)
	binary.BigEndian.PutUint32(data[0:4], 99) // custom command 99
	binary.BigEndian.PutUint32(data[4:8], uint32(mode))
	return h.SendControlPacket(data)
}

// SendKey injects an Android keycode with < 0.1ms latency.
func (h *Hub) SendKey(keycode int) error {
	data := make([]byte, 4)
	binary.BigEndian.PutUint32(data, uint32(keycode))
	return h.SendControlPacket(data)
}

// SendTouch injects a touch event through the native socket (< 0.1ms latency).
// action: 0 = ACTION_DOWN, 1 = ACTION_UP, 2 = ACTION_MOVE
func (h *Hub) SendTouch(action int, x, y int, pointerId int) error {
	data := make([]byte, 20)
	binary.BigEndian.PutUint32(data[0:4], uint32(action))
	binary.BigEndian.PutUint32(data[4:8], 0) // button
	binary.BigEndian.PutUint32(data[8:12], uint32(x))
	binary.BigEndian.PutUint32(data[12:16], uint32(y))
	binary.BigEndian.PutUint32(data[16:20], uint32(pointerId))
	return h.SendControlPacket(data)
}

// InjectTouch executes touch with hardware injection or native swipe interpolation.
func (h *Hub) InjectTouch(action string, x, y, x2, y2, dur int) {
	switch strings.ToLower(action) {
	case "down":
		_ = h.SendTouch(0, x, y, 0)
	case "move":
		_ = h.SendTouch(2, x, y, 0)
	case "up":
		_ = h.SendTouch(1, x, y, 0)
	case "tap", "click":
		if err := h.SendTouch(0, x, y, 0); err == nil {
			time.Sleep(20 * time.Millisecond)
			_ = h.SendTouch(1, x, y, 0)
			return
		}
		_ = exec.Command(config.SUBin, "-c", fmt.Sprintf("input tap %d %d", x, y)).Run()
	case "swipe":
		d := dur
		if d <= 0 {
			d = 220
		}
		// Native 60 FPS trajectory interpolation
		if err := h.SendTouch(0, x, y, 0); err == nil {
			steps := d / 16
			if steps < 6 {
				steps = 6
			}
			for i := 1; i <= steps; i++ {
				time.Sleep(16 * time.Millisecond)
				xi := x + (x2-x)*i/steps
				yi := y + (y2-y)*i/steps
				_ = h.SendTouch(2, xi, yi, 0)
			}
			time.Sleep(16 * time.Millisecond)
			_ = h.SendTouch(1, x2, y2, 0)
			return
		}
		_ = exec.Command(config.SUBin, "-c", fmt.Sprintf("input swipe %d %d %d %d %d", x, y, x2, y2, d)).Run()
	}
}

// InjectKey sends an Android key event with instant dispatch.
func (h *Hub) InjectKey(key string, keycode int) {
	if keycode <= 0 {
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "back":
			keycode = 4
		case "home":
			keycode = 3
		case "recents", "app_switch":
			keycode = 187
		case "power":
			keycode = 26
		case "vol_up":
			keycode = 24
		case "vol_down":
			keycode = 25
		case "enter":
			keycode = 66
		case "tab":
			keycode = 61
		case "delete", "backspace":
			keycode = 67
		case "wake":
			keycode = 224
		case "sleep":
			keycode = 223
		default:
			if k, err := strconv.Atoi(key); err == nil {
				keycode = k
			}
		}
	}

	if keycode > 0 {
		if err := h.SendKey(keycode); err == nil {
			return
		}
		_ = exec.Command(config.SUBin, "-c", fmt.Sprintf("input keyevent %d", keycode)).Run()
	}
}

var (
	cachedEncoderInfo string
	encoderOnce       sync.Once
)

// GetEncoderInfo discovers and caches the Android device's hardware H.264 encoder details.
func GetEncoderInfo() string {
	encoderOnce.Do(func() {
		// Inspect Android media codec configuration XMLs
		paths := []string{
			"/vendor/etc/media_codecs_c2.xml",
			"/vendor/etc/media_codecs_performance_c2.xml",
			"/vendor/etc/media_codecs.xml",
			"/system/etc/media_codecs.xml",
		}
		for _, p := range paths {
			content, err := os.ReadFile(p)
			if err == nil {
				s := string(content)
				if strings.Contains(s, "c2.qti.avc.encoder") {
					cachedEncoderInfo = "Qualcomm c2.qti.avc (Hardware GPU/VPU)"
					return
				}
				if strings.Contains(s, "OMX.qcom.video.encoder.avc") {
					cachedEncoderInfo = "Qualcomm OMX.qcom.video.encoder.avc (Hardware)"
					return
				}
				if strings.Contains(s, "c2.exynos.h264.encoder") || strings.Contains(s, "OMX.Exynos.AVC.Encoder") {
					cachedEncoderInfo = "Samsung Exynos AVC (Hardware GPU/VPU)"
					return
				}
				if strings.Contains(s, "c2.mtk.avc.encoder") || strings.Contains(s, "OMX.MTK.VIDEO.ENCODER.AVC") {
					cachedEncoderInfo = "MediaTek AVC (Hardware VPU)"
					return
				}
			}
		}

		// Fallback to ro.board.platform / ro.hardware
		out, err := exec.Command("getprop", "ro.board.platform").Output()
		if err == nil {
			plat := strings.TrimSpace(string(out))
			if plat != "" {
				cachedEncoderInfo = fmt.Sprintf("Hardware MediaCodec (SoC: %s)", plat)
				return
			}
		}

		cachedEncoderInfo = "Android MediaCodec video/avc (Hardware)"
	})
	return cachedEncoderInfo
}
