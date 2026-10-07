package logger

import (
	"testing"
)

func TestLoggerLevelsAndEntries(t *testing.T) {
	l := Get()
	l.Clear()

	l.Infof("network", "network initialized: %s", "eth0")
	l.Warnf("power", "battery low: %d%%", 15)
	l.Errorf("modem", "at port unreachable")

	entries := l.GetEntries(10)
	if len(entries) < 3 {
		t.Fatalf("expected at least 3 entries, got %d", len(entries))
	}

	netEntries := l.GetEntriesByCategory("network", 10)
	if len(netEntries) == 0 {
		t.Fatal("expected at least 1 network entry")
	}
	if netEntries[0].Category != "network" {
		t.Errorf("expected category 'network', got %s", netEntries[0].Category)
	}

	// Test ring buffer limit
	for i := range 150 {
		l.Infof("test", "message %d", i)
	}

	limited := l.GetEntries(5)
	if len(limited) != 5 {
		t.Errorf("expected 5 limited entries, got %d", len(limited))
	}

	l.Clear()
	cleared := l.GetEntries(10)
	if len(cleared) != 0 {
		t.Errorf("expected 0 entries after Clear, got %d", len(cleared))
	}
}

func TestLogWriter(t *testing.T) {
	w := &LogWriter{}
	data := []byte("daemon background log event")
	n, err := w.Write(data)
	if err != nil {
		t.Fatalf("unexpected error writing to LogWriter: %v", err)
	}
	if n != len(data) {
		t.Errorf("expected %d bytes written, got %d", len(data), n)
	}
}
