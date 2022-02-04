package bitmaps

import (
	"github.com/customerio/bitmaps/fixed"
)

type Bitmaps struct {
	sz Size
	b  []*fixed.Bitmap
}

type Size struct {
	Bits   uint32
	Chunks int
}

func New(sz Size) *Bitmaps {
	return &Bitmaps{
		sz: sz,
		b:  make([]*fixed.Bitmap, sz.Chunks),
	}
}

func (b *Bitmaps) Size() Size {
	return b.sz
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

func (b *Bitmaps) GetBitmaps() []*fixed.Bitmap {
	// TODO:
	// - Return a clone?
	// - Do we need to add all the empty bitmaps?
	return b.b
}

// And computes the intersection between two bitmaps and stores the result in the current bitmap.
func (b *Bitmaps) And(o *Bitmaps) {
	if len(b.b) != len(o.b) {
		panic("cannot AND two bitmaps with different cardinality")
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
		panic("cannot OR two bitmaps with different cardinality")
	}
	for i := 0; i < len(b.b); i++ {
		if b.b[i] == nil && o.b[i] != nil {
			b.b[i] = fixed.NewBitmap(int(b.sz.Bits))
		}
		if o.b[i] != nil {
			b.b[i].Or(o.b[i])
		}
	}
}

// Or computes the union between two bitmaps and stores the result in the current bitmap.
func (b *Bitmaps) AndNot(o *Bitmaps) {
	if len(b.b) != len(o.b) {
		panic("cannot AND NOT two bitmaps with different cardinality")
	}
	for i := 0; i < len(b.b); i++ {
		if b.b[i] != nil && o.b[i] != nil {
			b.b[i].AndNot(o.b[i])
		}
	}
}

// Flip negates the bits in the entire bitmap. Any integer present in the bitmap is removed, and
// any integer present in the bitmap is added.
func (b *Bitmaps) Flip() {
	for i := 0; i < len(b.b); i++ {
		if b.b[i] == nil {
			b.b[i] = fixed.NewBitmap(int(b.sz.Bits))
		}
		b.b[i].FlipInt(0, int(b.sz.Bits))
	}
}

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
	chunk := v / b.sz.Bits
	offset := v % b.sz.Bits
	if b.b[chunk] == nil {
		b.b[chunk] = fixed.NewBitmap(int(b.sz.Bits))
	}
	b.b[chunk].Add(offset)
}

// AddInt adds the integer x to the bitmap (convenience method: the parameter is casted to uint32 and we call Add).
func (b *Bitmaps) AddInt(v int) {
	b.Add(uint32(v))
}

// Remove the integer x from the bitmap.
func (b *Bitmaps) Remove(v uint32) {
	chunk := v / b.sz.Bits
	offset := v % b.sz.Bits
	if b.b[chunk] != nil {
		b.b[chunk].Remove(offset)
	}
}

// Contains returns true if the integer is contained in the bitmap.
func (b *Bitmaps) Contains(v uint32) bool {
	chunk := v / b.sz.Bits
	offset := v % b.sz.Bits
	if b.b[chunk] != nil {
		return b.b[chunk].Contains(offset)
	}
	return false
}

// ToArray creates a new slice containing all of the integers stored in the Bitmap in sorted order
func (b *Bitmaps) ToArray() []uint32 {
	a := []uint32{}
	for chunk, bits := range b.b {
		if bits != nil {
			arr := bits.ToArray()
			for _, v := range arr {
				a = append(a, uint32(chunk)*b.sz.Bits+v)
			}
		}
	}
	return a
}

// EachBatch calls process for each of the integers stored in the Bitmap.
// If process returns true, the iteration stops.
func (b *Bitmaps) EachBatch(batchSize int, process func([]uint32) (bool, error)) error {
	buf := make([]uint32, 0, batchSize)
	offset := uint32(0)
	for {
		buf, _ = b.NextMany(offset, buf, batchSize)
		if len(buf) == 0 {
			break
		}
		if done, err := process(buf); err != nil {
			return err
		} else if done {
			return nil
		}
		offset = buf[len(buf)-1] + 1
		buf = buf[:0]
	}

	return nil
}

// NextMany appends many next bit sets from the specified index,
// including possibly the current index and up to limit.
// If more is true, there are additional bits to be added.
//
//    buffer := uint32{}
//    j := uint32(0)
//	  for {
//		  var more bool
//		  buf, more = v.NextMany2(j, buf, 10)
//		  if !more {
//			  break
//		  }
//        do something with buf
//        buf = buf[:0] // possible clear buffer
//		  j = buf[len(buf)-1] + 1
//	}
//
// It is possible to retrieve all set bits as follow:
//
//    indices := make([]uint32, 0, bitmap.Count())
//    bitmap.NextMany2(0, indices, bitmap.Count())
//
// However if bitmap.Count() is large, it might be preferable to
// use several calls to NextMany2, for performance reasons.
func (b *Bitmaps) NextMany(i uint32, buffer []uint32, limit int) ([]uint32, bool) {
	chunk := uint32(i) / b.sz.Bits
	offset := uint32(i) % b.sz.Bits

	size := 0
	buf := make([]uint32, 0, limit)
	for int(chunk) < len(b.b) {
		if b.b[chunk] != nil {
			buf, _ = b.b[chunk].NextMany(offset, buf, limit-size)
			size += len(buf)
			if len(buf) > 0 {
				for _, v := range buf {
					buffer = append(buffer, chunk*b.sz.Bits+v)
				}
			}
			buf = buf[:0]
			if size == limit {
				return buffer, true
			}
		}
		chunk++
		offset = 0
	}
	return buffer, false
}

// NextManyInt is a helper for NextManyInt to operate on int not uint32.
func (b *Bitmaps) NextManyInt(i int, buffer []int, limit int) ([]int, bool) {
	chunk := uint32(i) / b.sz.Bits
	offset := uint32(i) % b.sz.Bits

	size := 0
	buf := make([]uint32, 0, limit)
	for int(chunk) < len(b.b) {
		if b.b[chunk] != nil {
			buf, _ = b.b[chunk].NextMany(offset, buf, limit-size)
			size += len(buf)
			if len(buf) > 0 {
				for _, v := range buf {
					buffer = append(buffer, int(chunk*b.sz.Bits+v))
				}
			}
			buf = buf[:0]
			if size == limit {
				return buffer, true
			}
		}
		chunk++
		offset = 0
	}
	return buffer, false
}

// Range returns all bits set after the start index to the stop index.
// index 0 1 2 3 4 5
// bit   0 2 4 6 8 10
// Range(2,3) == 2,4
func (m *Bitmaps) Range(start, stop int) []uint32 {
	offsets := make([]uint32, 0, stop-start)
	var cur int

	for chunk, b := range m.b {
		if b == nil {
			continue
		}
		// If our current position is less then the start position
		// for the results we want to return, move past this bitmap
		if card := int(b.GetCardinality()); cur+card < start {
			cur += card
			continue
		}

		// If our current position is past the stop position for the
		// results we want to return, we can stop iterating through bitmaps
		if cur >= stop {
			break
		}

		for _, offset := range b.ToArray() {
			// if our current position is in the range, add it to our result set
			if cur >= start && cur < stop {
				offsets = append(offsets, uint32(chunk)*m.sz.Bits+offset)
			}
			cur++
		}
	}

	return offsets
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
