package fixed

import (
	"errors"
	"fmt"
	"math/bits"
	"reflect"
	"unsafe"
)

// Note that the marshaled form of the bitmap is not portable -- it is assumed to be the
// same endianness as the machine that created the marshaled form
var (
	arrayMax = 1000

	headerSize = 8

	bitmapMagic    = uint32(0xFAD4F00D)
	encodingBitmap = byte(0xF0)
	encodingArray  = byte(0x0F)
)

// We're not going to range check here as we'd rather have a crash than a silent corruption.

const (
	// the wordSize of a bit set
	wordSize = 64

	// log2WordSize is lg(wordSize)
	log2WordSize = 6
)

type Bitmap struct {
	// Underlying storage for the header and the bitset.
	// header bytes | bits
	buf         []byte
	set         []uint64
	cardinality int
	nbits       int
}

// NewBitmap returns a fixed size bitmap with a capacity for nbits of storage.
func NewBitmap(nbits int) *Bitmap {
	totalSize := totalSize(nbits)
	buf := make([]byte, totalSize)
	return &Bitmap{
		buf:         buf,
		set:         toUint64Slice(buf[headerSize:]),
		cardinality: 0,
		nbits:       nbits,
	}
}

// NewBitmapFromBuf returns a fixed size bitmap with a capacity for nbits of storage.
// The bitmap is initialized from the marshaled form. If copyBuffer is true, the buffer
// is copied, otherwise it may be used by the bitmap itself.
func NewBitmapFromBuf(buf []byte, nbits int, copyBuffer bool) (*Bitmap, error) {
	if len(buf) < headerSize {
		return nil, errors.New("invalid data")
	}
	var h header
	h.read(buf)
	if h.magic != bitmapMagic {
		return nil, errors.New("bad magic")
	}

	totalSize := totalSize(nbits)
	switch h.encoding {
	case encodingBitmap:
		if len(buf) != totalSize {
			return nil, fmt.Errorf("bitmap expects %d bytes", totalSize)
		}
		if copyBuffer {
			dst := make([]byte, totalSize)
			copy(dst, buf)
			buf = dst
		}

		return &Bitmap{
			buf:         buf,
			set:         toUint64Slice(buf[headerSize:]),
			cardinality: int(h.cardinality),
			nbits:       nbits,
		}, nil

	case encodingArray:
		b := NewBitmap(nbits)
		if len(buf[headerSize:])/2 != int(h.cardinality) {
			return nil, fmt.Errorf("array encoding expects %d bytes", h.cardinality*2)
		}
		if h.cardinality > 0 {
			data := toUint16Slice(buf[headerSize:], int(h.cardinality))
			for _, v := range data {
				b.Add(uint32(v))
			}
		}
		return b, nil
	}

	return nil, fmt.Errorf("bad encoding")
}

// Bytes returns a pointer to the content of the bitmap.
func (b *Bitmap) Bytes() []byte {
	var header = header{
		magic:       bitmapMagic,
		encoding:    encodingBitmap,
		cardinality: uint16(b.GetCardinality()),
	}
	header.write(b.buf)
	return b.buf
}

// Marshal returns a binary encoding of the bitmap. The data
// returned may point to the internals of the bitmap itself,
// and if the bitmap is subsequently changed the marshaled form
// may change.
func (b *Bitmap) Marshal() ([]byte, error) {
	l := int(b.GetCardinality())

	if l >= arrayMax {
		return b.Bytes(), nil
	}

	buf := make([]byte, headerSize+l*2)
	var header = header{
		magic:       bitmapMagic,
		encoding:    encodingArray,
		cardinality: uint16(l),
	}
	header.write(buf)
	if l > 0 {
		data := toUint16Slice(buf[headerSize:], l)
		b.nextSetMany16(data)
	}
	return buf, nil
}

// Clone creates a copy of the bitmap.
func (b *Bitmap) Clone() *Bitmap {
	b1 := NewBitmap(b.nbits)
	copy(b1.set, b.set)
	b1.cardinality = b.cardinality
	return b1
}

// And computes the intersection between two bitmaps and stores the result in the current bitmap.
func (b *Bitmap) And(o *Bitmap) {
	l := len(o.set)
	cnt := 0
	for i := 0; i < l; i++ {
		v := b.set[i] & o.set[i]
		cnt += bits.OnesCount64(v)
		b.set[i] = v
	}
	b.cardinality = int(cnt)
}

// Or computes the union between two bitmaps and stores the result in the current bitmap.
func (b *Bitmap) Or(o *Bitmap) {
	l := len(o.set)
	cnt := 0
	for i := 0; i < l; i++ {
		v := b.set[i] | o.set[i]
		cnt += bits.OnesCount64(v)
		b.set[i] = v
	}
	b.cardinality = int(cnt)
}

// Or computes the union between two bitmaps and stores the result in the current bitmap.
func (b *Bitmap) AndNot(o *Bitmap) {
	l := len(o.set)
	cnt := 0
	for i := 0; i < l; i++ {
		v := b.set[i] &^ o.set[i]
		cnt += bits.OnesCount64(v)
		b.set[i] = v
	}
	b.cardinality = int(cnt)
}

// Flip negates the bits in the given range (i.e., [start,stop)), any integer present in this
// range and in the bitmap is removed, and any integer present in the range and not in the bitmap is added.
func (b *Bitmap) FlipInt(start, stop int) {
	if start >= stop {
		return
	}
	startWord := start >> log2WordSize
	endWord := stop >> log2WordSize
	b.set[startWord] ^= ^(^uint64(0) << (start & (wordSize - 1)))
	for i := startWord; i < endWord; i++ {
		b.set[i] = ^b.set[i]
	}
	b.set[endWord] ^= ^uint64(0) >> (-stop & (wordSize - 1))
	b.cardinality = int(b.computeCardinality())
}

// Equals returns true if the two bitmaps are the same, false otherwise.
func (b *Bitmap) Equals(o *Bitmap) bool {
	if o == nil && b == nil {
		return true
	}
	if o == nil && b != nil {
		return false
	}
	if o != nil && b == nil {
		return false
	}
	if b.nbits != o.nbits {
		return false
	}
	if b.cardinality != o.cardinality {
		return false
	}
	l := len(o.set)
	for i := 0; i < l; i++ {
		if b.set[i] != o.set[i] {
			return false
		}
	}
	return true
}

// GetCardinality returns the number of integers contained in the bitmap.
func (b *Bitmap) GetCardinality() uint64 {
	return uint64(b.cardinality)
}

// IsEmpty returns true if the Bitmap is empty.
func (b *Bitmap) IsEmpty() bool {
	return b.cardinality == 0
}

// Add the integer x to the bitmap.
func (b *Bitmap) Add(v uint32) {
	idx := v >> log2WordSize
	previous := b.set[idx]
	mask := uint64(1 << (v & (wordSize - 1)))
	newb := previous | mask
	b.set[idx] = newb
	b.cardinality += int((previous ^ newb) >> (v & (wordSize - 1)))
}

// AddInt adds the integer x to the bitmap (convenience method: the parameter is casted to uint32 and we call Add).
func (b *Bitmap) AddInt(v int) {
	b.Add(uint32(v))
}

// Remove the integer x from the bitmap.
func (b *Bitmap) Remove(v uint32) {
	if b.Contains(v) {
		b.cardinality--
		b.set[v>>log2WordSize] &^= 1 << (v & (wordSize - 1))
	}
}

// Contains returns true if the integer is contained in the bitmap.
func (b *Bitmap) Contains(v uint32) bool {
	// We're not going to range check here as we'd rather have
	// a crash than a silent corruption.
	//if int(v) >= b.nbits {
	//return false
	//}
	return b.set[v>>log2WordSize]&(1<<(v&(wordSize-1))) != 0
}

// ToArray creates a new slice containing all of the integers stored in the Bitmap in sorted order
func (b *Bitmap) ToArray() []uint32 {
	indices := make([]uint32, b.GetCardinality())
	b.nextSetMany32(indices)
	return indices
}

func (b *Bitmap) computeCardinality() uint64 {
	cnt := 0
	for _, x := range b.set {
		cnt += bits.OnesCount64(x)
	}
	return uint64(cnt)
}

func (b *Bitmap) nextSetMany16(buffer []uint16) {
	myanswer := buffer
	capacity := cap(buffer)
	x := 0
	if x >= len(b.set) || capacity == 0 {
		return
	}
	size := int(0)
	for idx, word := range b.set {
		for word != 0 {
			r := bits.TrailingZeros64(word)
			t := word & ((^word) + 1)
			myanswer[size] = uint16(r + (x+idx)<<6)
			size++
			if size == capacity {
				return
			}
			word = word ^ t
		}
	}
}

func (b *Bitmap) nextSetMany32(buffer []uint32) {
	myanswer := buffer
	capacity := cap(buffer)
	x := 0
	if x >= len(b.set) || capacity == 0 {
		return
	}
	size := int(0)
	for idx, word := range b.set {
		for word != 0 {
			r := bits.TrailingZeros64(word)
			t := word & ((^word) + 1)
			myanswer[size] = uint32(r + (x+idx)<<6)
			size++
			if size == capacity {
				return
			}
			word = word ^ t
		}
	}
}

// And computes the intersection between the bitmaps and returns the result.
func AndBitmaps(bitmaps ...*Bitmap) *Bitmap {
	if len(bitmaps) == 0 {
		return NewBitmap(0)
	}
	b := bitmaps[0].Clone()
	for _, o := range bitmaps[1:] {
		b.And(o)
	}
	return b
}

// Or computes the union between the bitmaps and stores the returns the result.
func OrBitmaps(bitmaps ...*Bitmap) *Bitmap {
	if len(bitmaps) == 0 {
		return NewBitmap(0)
	}
	b := bitmaps[0].Clone()
	for _, o := range bitmaps[1:] {
		b.Or(o)
	}
	return b
}

// AndNot computes the difference between the bitmaps and returns the result.
func AndNotBitmap(a *Bitmap, b *Bitmap) *Bitmap {
	c := a.Clone()
	c.AndNot(b)
	return c
}

// FlipBitmap negates the bits in the the bitmaps range and returns the result.
func FlipBitmap(b *Bitmap, start, stop int) *Bitmap {
	c := b.Clone()
	c.FlipInt(start, stop)
	return c
}

func bodySize(nbits int) int {
	return headerSize + (8 * ((nbits / wordSize) + 1))
}

func totalSize(nbits int) int {
	return headerSize + bodySize(nbits)
}

// Data encoding.
// 64 bit header.
type header struct {
	magic       uint32 // magic uint32
	encoding    byte   // encoding uint8
	unused      byte   // unused uint8
	cardinality uint16 // cardinality uint16
}

func (h *header) read(buf []byte) {
	data := toUint64Slice(buf)
	v := data[0]
	h.magic = uint32((v & 0xFFFFFFFF00000000) >> 32)
	h.encoding = byte((v & 0xFF000000) >> 24)
	h.cardinality = uint16(v & 0xFFFF)
}

func (h header) write(buf []byte) {
	data := toUint64Slice(buf)
	data[0] = uint64(h.magic)<<32 | uint64(h.encoding)<<24 | uint64(h.cardinality)
}

func toUint64Slice(b []byte) []uint64 {
	// reference: https://go101.org/article/unsafe.html
	var u64s []uint64
	hdr := (*reflect.SliceHeader)(unsafe.Pointer(&u64s))
	hdr.Len = len(b) / 8
	hdr.Cap = hdr.Len
	hdr.Data = uintptr(unsafe.Pointer(&b[0]))
	return u64s
}

func toUint16Slice(b []byte, l int) []uint16 {
	var u16s []uint16
	hdr := (*reflect.SliceHeader)(unsafe.Pointer(&u16s))
	hdr.Len = l
	hdr.Cap = len(b) / 2
	hdr.Data = uintptr(unsafe.Pointer(&b[0]))
	return u16s
}
