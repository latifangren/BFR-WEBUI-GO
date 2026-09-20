package scrcpy_test

import (
	"sync"
	"testing"
	"time"

	"bfr-webui-go/internal/scrcpy"
	"github.com/gorilla/websocket"
)

func TestSanitizeInput(t *testing.T) {
	// Shell injection attempt with backticks, semicolons, and spaces
	rawInput := "hello world; rm -rf /; `whoami` $PATH"
	sanitized := scrcpy.SanitizeInput(rawInput)

	// Spaces replaced by %s, shell operators stripped
	expected := "hello%sworld%srm%s-rf%s/%swhoami%sPATH"
	if sanitized != expected {
		t.Errorf("expected sanitized string '%s', got '%s'", expected, sanitized)
	}

	// Empty string
	if empty := scrcpy.SanitizeInput(""); empty != "" {
		t.Errorf("expected empty string for empty input, got '%s'", empty)
	}
}

func TestWSClient_ConcurrentSendNoPanic(t *testing.T) {
	hub := scrcpy.GetHub()
	client := scrcpy.NewWSClient(hub, nil)
	defer client.Close()

	var wg sync.WaitGroup
	workers := 50
	messagesPerWorker := 100

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < messagesPerWorker; j++ {
				_ = client.Send(websocket.BinaryMessage, []byte{byte(id), byte(j)})
			}
		}(i)
	}

	wg.Wait()
}

func TestWSClient_SlowClientDropNonBlocking(t *testing.T) {
	hub := scrcpy.GetHub()
	client := scrcpy.NewWSClient(hub, nil)
	defer client.Close()

	// Fill the 64-slot channel buffer
	for i := 0; i < 64; i++ {
		ok := client.Send(websocket.BinaryMessage, []byte{byte(i)})
		if !ok {
			t.Fatalf("expected send #%d to succeed into empty buffer", i)
		}
	}

	// The 65th send should drop non-blockingly and return false
	start := time.Now()
	dropped := client.Send(websocket.BinaryMessage, []byte("overflow_frame"))
	elapsed := time.Since(start)

	if dropped {
		t.Errorf("expected 65th send to be dropped (return false), but got true")
	}
	if elapsed > 10*time.Millisecond {
		t.Errorf("send blocked for %v, expected non-blocking immediate return", elapsed)
	}
}

func TestWSClient_CloseIdempotent(t *testing.T) {
	hub := scrcpy.GetHub()
	client := scrcpy.NewWSClient(hub, nil)

	// Multiple closes should not panic
	client.Close()
	client.Close()
	client.Close()

	// Send after close should return false
	if ok := client.Send(websocket.BinaryMessage, []byte("after close")); ok {
		t.Errorf("expected Send() after Close() to return false")
	}
}

func TestHub_RegisterUnregisterLifecycle(t *testing.T) {
	hub := scrcpy.GetHub()

	// RegisterClient with nil conn returns valid WSClient without crashing
	client, _, _, _, _, _ := hub.RegisterClient(nil)
	if client == nil {
		t.Fatalf("expected non-nil WSClient from RegisterClient")
	}

	// Verify send works
	ok := client.Send(websocket.TextMessage, []byte("hello"))
	if !ok {
		t.Errorf("expected send to newly registered client to succeed")
	}

	// Unregister client
	hub.UnregisterClient(client)

	// Sending after unregister/close should safely fail
	if ok := client.Send(websocket.TextMessage, []byte("after unregister")); ok {
		t.Errorf("expected send after unregister to return false")
	}
}
