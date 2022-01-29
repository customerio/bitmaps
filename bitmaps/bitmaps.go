package bitmaps

import (
	"github.com/customerio/bitmaps/fixed"
)

type Bitmaps struct {
	sz Size
	b  []*fixed.Bitmap
}

type Size struct {
	nbits   uint32
	nchunks int
}

func New(sz Size) *Bitmaps {
	return &Bitmaps{
		sz: sz,
		b:  make([]*fixed.Bitmap, sz.nchunks),
	}
}

func (b *Bitmaps) Set(chunk uint32, bits *fixed.Bitmap) {
	b.b[chunk] = bits
}

// Clone creates a copy of the bitmap.
func (b *Bitmaps) Clone() *Bitmaps {
	b1 := New(b.sz)
	for i := 0; i < len(b.b); i++ {
		if b.b[i] != nil {
			b1.b[i] = b.b[i].Clone()
		}
	}
	return b1
}

// And computes the intersection between two bitmaps and stores the result in the current bitmap.
func (b *Bitmaps) And(o *Bitmaps) {
	if len(b.b) != len(o.b) {
		return
	}
	for i := 0; i < len(b.b); i++ {
		if b.b[i] == nil {
			o.b[i] = nil
		} else if o.b[i] == nil {
			b.b[i] = nil
		} else {
			b.b[i].And(o.b[i])
		}
	}
}

// Or computes the union between two bitmaps and stores the result in the current bitmap.
func (b *Bitmaps) Or(o *Bitmaps) {
	if len(b.b) != len(o.b) {
		return
	}
	for i := 0; i < len(b.b); i++ {
		if b.b[i] == nil && o.b[i] != nil {
			b.b[i] = fixed.NewBitmap(int(b.sz.nbits))
		}
		if o.b[i] != nil {
			b.b[i].Or(o.b[i])
		}
	}
}

// Or computes the union between two bitmaps and stores the result in the current bitmap.
func (b *Bitmaps) AndNot(o *Bitmaps) {
}

// Flip negates the bits in the given range (i.e., [start,stop)), any integer present in this
// range and in the bitmap is removed, and any integer present in the range and not in the bitmap is added.
/*
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
*/

// Equals returns true if the two bitmaps are the same, false otherwise.
func (b *Bitmaps) Equals(o *Bitmaps) bool {
	if o == nil && b == nil {
		return true
	}
	if o == nil && b != nil {
		return false
	}
	if len(b.b) != len(o.b) {
		return false
	}
	for i := 0; i < len(b.b); i++ {
		if !b.b[i].Equals(o.b[i]) {
			return false
		}
	}
	return true
}

// GetCardinality returns the number of integers contained in the bitmap.
func (b *Bitmaps) GetCardinality() uint64 {
	c := uint64(0)
	for _, b := range b.b {
		if b != nil {
			c += b.GetCardinality()
		}
	}
	return c
}

// IsEmpty returns true if the Bitmap is empty.
func (b *Bitmaps) IsEmpty() bool {
	for _, b := range b.b {
		if b != nil && !b.IsEmpty() {
			return false
		}
	}
	return true
}

// Add the integer x to the bitmap.
func (b *Bitmaps) Add(v uint32) {
	chunk := v / b.sz.nbits
	offset := v % b.sz.nbits
	if b.b[chunk] == nil {
		b.b[chunk] = fixed.NewBitmap(int(b.sz.nbits))
	}
	b.b[chunk].Add(offset)
}

// AddInt adds the integer x to the bitmap (convenience method: the parameter is casted to uint32 and we call Add).
func (b *Bitmaps) AddInt(v int) {
	b.Add(uint32(v))
}

// Remove the integer x from the bitmap.
func (b *Bitmaps) Remove(v uint32) {
	chunk := v / b.sz.nbits
	offset := v % b.sz.nbits
	if b.b[chunk] != nil {
		b.b[chunk].Remove(offset)
	}
}

// Contains returns true if the integer is contained in the bitmap.
func (b *Bitmaps) Contains(v uint32) bool {
	chunk := v / b.sz.nbits
	offset := v % b.sz.nbits
	if b.b[chunk] != nil {
		return b.b[chunk].Contains(offset)
	}
	return false
}

// ToArray creates a new slice containing all of the integers stored in the Bitmap in sorted order
func (b *Bitmaps) ToArray() []uint32 {
	a := []uint32{}
	for chunk, bits := range b.b {
		arr := bits.ToArray()
		for _, v := range arr {
			a = append(a, uint32(chunk)*b.sz.nbits+v)
		}
	}
	return a
}

// And computes the intersection between the bitmaps and returns the result.
func AndBitmaps(sz Size, bitmaps ...*Bitmaps) *Bitmaps {
	if len(bitmaps) == 0 {
		return New(sz)
	}
	b := bitmaps[0].Clone()
	for _, o := range bitmaps[1:] {
		b.And(o)
	}
	return b
}

// Or computes the union between the bitmaps and stores the returns the result.
func OrBitmaps(sz Size, bitmaps ...*Bitmaps) *Bitmaps {
	if len(bitmaps) == 0 {
		return New(sz)
	}
	b := bitmaps[0].Clone()
	for _, o := range bitmaps[1:] {
		b.Or(o)
	}
	return b
}

// AndNot computes the difference between the bitmaps and returns the result.
func AndNotBitmap(a *Bitmaps, b *Bitmaps) *Bitmaps {
	c := a.Clone()
	c.AndNot(b)
	return c
}

/*
// FlipBitmap negates the bits in the the bitmaps range and returns the result.
func FlipBitmap(b *Bitmaps, start, stop int) *Bitmaps {
	c := b.Clone()
	c.FlipInt(start, stop)
	return c
}
*/