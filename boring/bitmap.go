package boring

import (
	"errors"
	"fmt"
	"reflect"
	"unsafe"
)

// Note that the marshaled form of the bitmap is not portable -- it is assumed to be the
// same endianness as the machine that created the marshaled form
var (
	// Memory is layed out as follows:
	// header | data
	// The data block is a fixed size block of memory with enough space to old nbits of storage.
	// For bitmaps:
	// 8 bytes| []uint64 bits
	// For arrays
	// 8 bytes| []uint16 contents
	headerSize = 8

	// For storing uint16 the buffer has capacity to store 1,875 uint16 (30k/16)

	bitmapMagic    = uint32(0xFAD4F00D)
	encodingBitmap = byte(0xF0)
	encodingArray  = byte(0x0F)
)

type Bitmap struct {
	buf      []byte
	encoding byte
	nbits    int
	array    array
	bitmap   bitmap
}

// NewBitmap returns a fixed size bitmap with a capacity for nbits of storage.
func NewBitmap(nbits int) *Bitmap {
	totalSize := totalSize(nbits)
	buf := make([]byte, totalSize)
	return &Bitmap{
		buf:      buf,
		nbits:    nbits,
		encoding: encodingArray,
		array: array{
			buf:     buf,
			content: toUint16Slice(buf[headerSize:], 0),
			// once we go over this limit, we'll change to a bitmap.
			sz: bodySize(nbits) / (16 * 2),
		},
		bitmap: bitmap{
			buf:         buf,
			set:         toUint64Slice(buf[headerSize:]),
			cardinality: 0,
		},
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
		if copyBuffer {
			dst := make([]byte, totalSize)
			copy(dst, buf)
			buf = dst
		}
		if len(buf) != totalSize {
			return nil, fmt.Errorf("bitmap expects %d bytes", totalSize)
		}
		return &Bitmap{
			buf:      buf,
			encoding: encodingBitmap,
			nbits:    nbits,
			array: array{
				buf:     buf,
				content: toUint16Slice(buf[headerSize:], 0),
				// once we go over this limit, we'll change to a bitmap.
				sz: bodySize(nbits) / (16 * 2),
			},
			bitmap: bitmap{
				buf:         buf,
				set:         toUint64Slice(buf[headerSize:]),
				cardinality: int(h.cardinality),
			},
		}, nil

	case encodingArray:
		dst := make([]byte, totalSize)
		copy(dst, buf)
		buf = dst

		return &Bitmap{
			buf:      buf,
			nbits:    nbits,
			encoding: encodingArray,
			array: array{
				buf:     buf,
				content: toUint16Slice(buf[headerSize:], int(h.cardinality)),
				// once we go over this limit, we'll change to a bitmap.
				sz: bodySize(nbits) / (16 * 2),
			},
			bitmap: bitmap{
				buf:         buf,
				set:         toUint64Slice(buf[headerSize:]),
				cardinality: 0,
			},
		}, nil
	}
	return nil, fmt.Errorf("bad encoding")
}

// Bytes returns a pointer to the content of the bitmap.
func (b *Bitmap) Bytes() []byte {
	var header = header{
		magic:       bitmapMagic,
		encoding:    b.encoding,
		cardinality: uint16(b.GetCardinality()),
	}
	header.write(b.buf)
	buf := b.buf
	if b.encoding == encodingArray {
		buf = buf[:headerSize+len(b.array.content)*2]
	}
	return buf
}

// Marshal returns a binary encoding of the bitmap. The data
// returned may point to the internals of the bitmap itself,
// and if the bitmap is subsequently changed the marshaled form
// may change.
func (b *Bitmap) Marshal() ([]byte, error) {
	return b.Bytes(), nil
}

// Bytes returns a pointer to the content of the bitmap.
func (b *Bitmap) Clone() *Bitmap {
	c, _ := NewBitmapFromBuf(b.Bytes(), b.nbits, true)
	return c
}

// Add the integer x to the bitmap.
func (b *Bitmap) Add(v uint32) {
	if b.encoding == encodingArray {
		b.array.add(v)
		b.convertMaybe()
	} else {
		b.bitmap.add(v)
	}
}

// AddInt adds the integer x to the bitmap (convenience method: the parameter is casted to uint32 and we call Add).
func (b *Bitmap) AddInt(v int) {
	b.Add(uint32(v))
}

// Remove the integer x from the bitmap.
func (b *Bitmap) Remove(v uint32) {
	if b.encoding == encodingArray {
		b.array.remove(v)
	} else {
		b.bitmap.remove(v)
		b.convertMaybe()
	}
}

// Contains returns true if the integer is contained in the bitmap.
func (b *Bitmap) Contains(v uint32) bool {
	if b.encoding == encodingArray {
		return b.array.contains(v)
	} else {
		return b.bitmap.contains(v)
	}
}

func (b *Bitmap) convertMaybe() {
	switch b.encoding {
	case encodingArray:
		if len(b.array.content) >= b.array.sz {
			b.convertEncoding(encodingBitmap)
		}
	case encodingBitmap:
		if b.bitmap.cardinality < b.array.sz {
			b.convertEncoding(encodingArray)
		}
	}
}

// And computes the intersection between two bitmaps and stores the result in the current bitmap.
func (b *Bitmap) And(o *Bitmap) {
	if b == o || o == nil {
		return
	}
	if b.nbits != o.nbits {
		return
	}
	if b.encoding == encodingArray {
		if o.encoding == encodingArray {
			b.array.and(o.array)
		} else {
			b.array.andBitmap(o.bitmap)
		}
	} else {
		if o.encoding == encodingArray {
			o.convertEncoding(encodingBitmap)
			b.bitmap.and(o.bitmap)
			o.convertMaybe()
		} else {
			b.bitmap.and(o.bitmap)
		}
	}
	b.convertMaybe()
}

// Or computes the union between two bitmaps and stores the result in the current bitmap.
func (b *Bitmap) Or(o *Bitmap) {
	if b == o || o == nil {
		return
	}
	if b.nbits != o.nbits {
		return
	}
	if b.encoding == encodingArray {
		if o.encoding == encodingArray {
			b.array.or(o.array)
		} else {
			b.convertEncoding(encodingBitmap)
			b.bitmap.or(o.bitmap)
		}
		b.convertMaybe()
	} else {
		if o.encoding == encodingArray {
			for _, v := range o.array.content {
				b.Add(uint32(v))
			}
		} else {
			b.bitmap.or(o.bitmap)
		}
	}
}

// Or computes the union between two bitmaps and stores the result in the current bitmap.
func (b *Bitmap) AndNot(o *Bitmap) {
	if b.nbits != o.nbits {
		return
	}
	if b.encoding == encodingArray {
		if o.encoding == encodingArray {
			b.array.andNot(o.array)
		} else {
			b.array.andNotBitmap(o.bitmap)
		}
	} else {
		if o.encoding == encodingArray {
			b.bitmap.andNotArray(o.array)
		} else {
			b.bitmap.andNot(o.bitmap)
		}
	}
	b.convertMaybe()
}

// Flip negates the bits in the given range (i.e., [start,stop)), any integer present in this
// range and in the bitmap is removed, and any integer present in the range and not in the bitmap is added.
func (b *Bitmap) FlipInt(start, stop int) {
	if start >= stop {
		return
	}
	if start < 0 {
		start = 0
	}
	if stop >= b.nbits {
		stop = b.nbits - 1
	}
	if b.encoding == encodingArray {
		b.convertEncoding(encodingBitmap)
	}

	b.bitmap.flip(start, stop)
	b.convertMaybe()
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
	if b.GetCardinality() != o.GetCardinality() {
		return false
	}
	if b.encoding == encodingArray {
		if o.encoding == encodingArray {
			return b.array.equals(o.array)
		} else {
			return b.array.equalsBitmap(o.bitmap)
		}
	} else {
		if o.encoding == encodingArray {
			return b.bitmap.equalsArray(o.array)
		} else {
			return b.bitmap.equals(o.bitmap)
		}
	}
	return true
}

// GetCardinality returns the number of integers contained in the bitmap.
func (b *Bitmap) GetCardinality() uint64 {
	if b.encoding == encodingArray {
		return uint64(len(b.array.content))
	} else {
		return uint64(b.bitmap.cardinality)
	}
}

func (b *Bitmap) convertEncoding(encoding byte) {
	if b.encoding == encoding {
		return
	}
	data := make([]uint16, b.GetCardinality())
	if encoding == encodingArray {
		b.bitmap.nextSetMany16(data)
		b.array.content = toUint16Slice(b.buf[headerSize:], len(data))
		copy(b.array.content, data)
	} else {
		copy(data, b.array.content)
		// Clear memory (must clear the bitmap).
		for i := 0; i < len(b.buf); i++ {
			b.buf[i] = 0
		}
		b.bitmap.cardinality = 0
		for _, v := range data {
			b.bitmap.add(uint32(v))
		}
	}
	b.encoding = encoding
}

// IsEmpty returns true if the Bitmap is empty.
func (b *Bitmap) IsEmpty() bool {
	return b.GetCardinality() == 0
}

// ToArray creates a new slice containing all of the integers stored in the Bitmap in sorted order
func (b *Bitmap) ToArray() []uint32 {
	indices := make([]uint32, b.GetCardinality())
	if b.encoding == encodingArray {
		for i, v := range b.array.content {
			indices[i] = uint32(v)
		}
	} else {
		b.bitmap.nextSetMany32(indices)
	}
	return indices
}

// And computes the intersection between the bitmaps and returns the result.
func AndBitmaps(nbits int, bitmaps ...*Bitmap) *Bitmap {
	if len(bitmaps) == 0 {
		return NewBitmap(nbits)
	}
	b := bitmaps[0].Clone()
	for _, o := range bitmaps[1:] {
		b.And(o)
	}
	return b
}

// Or computes the union between the bitmaps and stores the returns the result.
func OrBitmaps(nbits int, bitmaps ...*Bitmap) *Bitmap {
	if len(bitmaps) == 0 {
		return NewBitmap(nbits)
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

func bodySize(nbits int) int {
	return headerSize + (8 * ((nbits / 64) + 1))
}

func totalSize(nbits int) int {
	return headerSize + bodySize(nbits)
}
