package bufferpool

import (
	"testing"
)

func TestGetAndPutBuffer(t *testing.T) {
	buf := GetBuffer()
	if buf == nil {
		t.Fatal("expected non-nil buffer from GetBuffer")
	}

	buf.WriteString("hello world")
	if buf.String() != "hello world" {
		t.Errorf("unexpected content: %s", buf.String())
	}

	PutBuffer(buf)
	// PutBuffer(nil) should not panic
	PutBuffer(nil)

	buf2 := GetBuffer()
	if buf2.Len() != 0 {
		t.Errorf("expected clean reset buffer, got length %d", buf2.Len())
	}
	PutBuffer(buf2)
}

func TestGetBytesAndPutBytes(t *testing.T) {
	testSizes := []int{512, 1024, 1500, 2048, 16384, 32768, 65536}

	for _, sz := range testSizes {
		b := GetBytes(sz)
		if len(b) != sz {
			t.Errorf("size %d: expected slice length %d, got %d", sz, sz, len(b))
		}
		for i := range b {
			b[i] = byte(i % 256)
		}

		// return to pool
		PutBytes(b)
	}

	// PutBytes with arbitrary capacity should not panic
	customSlice := make([]byte, 500)
	PutBytes(customSlice)
}
