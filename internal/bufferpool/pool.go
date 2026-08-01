package bufferpool

import (
	"bytes"
	"sync"
)

var bufferPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

// GetBuffer retrieves a reset bytes.Buffer from the buffer pool.
func GetBuffer() *bytes.Buffer {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

// PutBuffer resets and returns a bytes.Buffer to the buffer pool.
func PutBuffer(buf *bytes.Buffer) {
	if buf == nil {
		return
	}
	buf.Reset()
	bufferPool.Put(buf)
}

var slicePool1K = sync.Pool{
	New: func() interface{} {
		b := make([]byte, 1024)
		return &b
	},
}

var slicePool2K = sync.Pool{
	New: func() interface{} {
		b := make([]byte, 2048)
		return &b
	},
}

var slicePool32K = sync.Pool{
	New: func() interface{} {
		b := make([]byte, 32768)
		return &b
	},
}

// GetBytes retrieves a byte slice from pooled memory buckets based on requested size.
func GetBytes(size int) []byte {
	if size <= 1024 {
		bp := slicePool1K.Get().(*[]byte)
		return (*bp)[:size]
	}
	if size <= 2048 {
		bp := slicePool2K.Get().(*[]byte)
		return (*bp)[:size]
	}
	if size <= 32768 {
		bp := slicePool32K.Get().(*[]byte)
		return (*bp)[:size]
	}
	return make([]byte, size)
}

// PutBytes returns a byte slice to the slice pool if its capacity matches a pool bucket.
func PutBytes(b []byte) {
	capB := cap(b)
	if capB == 1024 {
		slice := b[:1024]
		slicePool1K.Put(&slice)
	} else if capB == 2048 {
		slice := b[:2048]
		slicePool2K.Put(&slice)
	} else if capB == 32768 {
		slice := b[:32768]
		slicePool32K.Put(&slice)
	}
}
