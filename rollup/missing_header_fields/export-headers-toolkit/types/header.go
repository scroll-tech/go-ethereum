package types

import (
	"encoding/binary"
	"fmt"
)

const HeaderSizeSerialized = 2

type Header struct {
	Number     uint64
	Difficulty uint64
	ExtraData  []byte
}

func (h *Header) String() string {
	return fmt.Sprintf("%d,%d,0x%x\n", h.Number, h.Difficulty, h.ExtraData)
}

// Bytes returns the byte representation of the header including the initial 2 bytes for the size.
func (h *Header) Bytes() ([]byte, error) {
	size := 8 + 8 + len(h.ExtraData)

	buf := make([]byte, HeaderSizeSerialized+size)
	binary.BigEndian.PutUint16(buf[:2], uint16(size))
	binary.BigEndian.PutUint64(buf[2:10], h.Number)
	binary.BigEndian.PutUint64(buf[10:18], h.Difficulty)
	copy(buf[18:], h.ExtraData)
	return buf, nil
}

func (h *Header) Vanity() [32]byte {
	return [32]byte(h.ExtraData[:32])
}

func (h *Header) SealLen() int {
	return len(h.ExtraData[32:])
}

// FromBytes reads the header from the byte representation excluding the initial 2 bytes for the size.
func (h *Header) FromBytes(buf []byte) *Header {
	h.Number = binary.BigEndian.Uint64(buf[:8])
	h.Difficulty = binary.BigEndian.Uint64(buf[8:16])
	h.ExtraData = buf[16:]

	return h
}

type HeaderHeap []*Header

func (h HeaderHeap) Len() int            { return len(h) }
func (h HeaderHeap) Less(i, j int) bool  { return h[i].Number < h[j].Number }
func (h HeaderHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *HeaderHeap) Push(x interface{}) { *h = append(*h, x.(*Header)) }
func (h *HeaderHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[0 : n-1]
	return item
}
