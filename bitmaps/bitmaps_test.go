package bitmaps

import (
	"testing"
)

var nbits = 30
var nchunks = 1000 // 30k bits of storage.

var sz = Size{uint32(nbits), nchunks}

func TestConvert(t *testing.T) {
	b := New(sz)
	for v := uint32(0); v < uint32(30000); v += 2 {
		b.Add(v)
	}
}

func TestClone(t *testing.T) {
	b := New(sz)
	for v := uint32(0); v < uint32(30000); v += 2 {
		b.Add(v)
	}
	b1 := New(sz)
	for v := uint32(1); v < uint32(30000); v += 2 {
		b1.Add(v)
	}
	for i := 0; i < 100; i++ {
		c := b.Clone()
		c.Or(b1)
		if b.GetCardinality() != 15000 {
			t.Error("Or failed")
			return
		}
		if c.GetCardinality() != 30000 {
			t.Error("Or failed")
			return
		}

		c = b1.Clone()
		c.Or(b)
		if b.GetCardinality() != 15000 {
			t.Error("Or failed")
			return
		}
		if c.GetCardinality() != 30000 {
			t.Error("Or failed")
			return
		}
	}
}

func TestOrShort(t *testing.T) {
	b := New(sz)
	for v := uint32(0); v < uint32(30000); v += 100 {
		o := New(sz)
		for vv := v; vv < v+100; vv++ {
			o.Add(vv)
		}
		b.Or(o)
	}
	if b.GetCardinality() != 30000 {
		t.Error("Or failed")
		return
	}
}

func TestEquals(t *testing.T) {
	a := New(sz)
	c := New(sz)
	if !a.Equals(c) {
		t.Error("Two empty sets of the same size should be equal")
		return
	}
	a.Add(99)
	c.Add(0)
	if a.Equals(c) {
		t.Error("Two sets with differences should not be equal")
		return
	}
	c.Add(99)
	a.Add(0)
	if !a.Equals(c) {
		t.Error("Two sets with the same bits set should be equal")
		return
	}
	if a.Equals(nil) {
		t.Error("The sets should be different")
		return
	}
}

func TestContains(t *testing.T) {
	a := New(sz)
	if a.Contains(99) {
		t.Error("bitmap should not contain 99")
		return
	}
	a.Add(99)
	if !a.Contains(99) {
		t.Error("bitmap should contain 99")
		return
	}
}

func TestRemove(t *testing.T) {
	a := New(sz)
	a.Add(99)
	a.Remove(99)
	if a.Contains(99) {
		t.Error("bitmap should not contain 99")
		return
	}
	if a.GetCardinality() != 0 {
		t.Error("bitmap should be empty")
		return
	}
	if !a.IsEmpty() {
		t.Error("bitmap should be empty")
		return
	}
}

func TestOrBitmaps(t *testing.T) {
	a := New(sz)
	b := New(sz)
	for i := uint32(1); i < 100; i += 2 {
		a.Add(i)
		b.Add(i - 1)
	}
	for i := uint32(100); i < 200; i++ {
		b.Add(i)
	}
	c := OrBitmaps(sz, a, b)
	d := OrBitmaps(sz, b, a)
	if c.GetCardinality() != 200 {
		t.Errorf("Union should have 200 bits set, but had %d", c.GetCardinality())
	}
	if !c.Equals(d) {
		t.Errorf("Union should be symmetric")
	}
}

func TestOr(t *testing.T) {
	a := New(sz)
	b := New(sz)
	for i := uint32(1); i < 100; i += 2 {
		a.Add(i)
		b.Add(i - 1)
	}
	for i := uint32(100); i < 200; i++ {
		b.Add(i)
	}
	c := a.Clone()
	c.Or(b)
	d := b.Clone()
	d.Or(a)
	if c.GetCardinality() != 200 {
		t.Errorf("Union should have 200 bits set, but had %d", c.GetCardinality())
	}
	if d.GetCardinality() != 200 {
		t.Errorf("Union should have 200 bits set, but had %d", d.GetCardinality())
	}
	if !c.Equals(d) {
		t.Errorf("Union should be symmetric")
	}
}

func TestAndBitmaps(t *testing.T) {
	a := New(sz)
	b := New(sz)
	for i := uint32(1); i < 100; i += 2 {
		a.Add(i)
		b.Add(i - 1)
		b.Add(i)
	}
	for i := uint32(100); i < 200; i++ {
		b.Add(i)
	}
	c := AndBitmaps(sz, a, b)
	d := AndBitmaps(sz, b, a)
	if c.GetCardinality() != 50 {
		t.Errorf("Intersection should have 50 bits set, but had %d", c.GetCardinality())
	}
	if !c.Equals(d) {
		t.Errorf("Intersection should be symmetric")
	}
}

func TestAnd(t *testing.T) {
	a := New(sz)
	b := New(sz)
	for i := uint32(1); i < 100; i += 2 {
		a.Add(i)
		b.Add(i - 1)
		b.Add(i)
	}
	for i := uint32(100); i < 200; i++ {
		b.Add(i)
	}
	c := a.Clone()
	c.And(b)
	d := b.Clone()
	d.And(a)
	if c.GetCardinality() != 50 {
		t.Errorf("Intersection should have 50 bits set, but had %d", c.GetCardinality())
	}
	if d.GetCardinality() != 50 {
		t.Errorf("Intersection should have 50 bits set, but had %d", d.GetCardinality())
	}
	if !c.Equals(d) {
		t.Errorf("Intersection should be symmetric")
	}
}

func TestAndNotBitmaps(t *testing.T) {
	a := New(sz)
	b := New(sz)
	for i := uint32(0); i < 50; i++ {
		a.Add(i)
	}
	for i := uint32(50); i < 150; i++ {
		b.Add(i)
	}
	for i := uint32(100); i < 150; i++ {
		a.Add(i)
	}

	c := AndNotBitmap(a, b)
	d := AndNotBitmap(b, a)
	if c.GetCardinality() != 50 {
		t.Errorf("a-b Difference should have 50 bits set, but had %d", c.GetCardinality())
	}
	if d.GetCardinality() != 50 {
		t.Errorf("b-a Difference should have 150 bits set, but had %d", d.GetCardinality())
	}
	if c.Equals(d) {
		t.Errorf("Difference, here, should not be symmetric")
	}
}

func TestAndNot(t *testing.T) {
	a := New(sz)
	b := New(sz)
	for i := uint32(0); i < 50; i++ {
		a.Add(i)
	}
	for i := uint32(50); i < 150; i++ {
		b.Add(i)
	}
	for i := uint32(100); i < 150; i++ {
		a.Add(i)
	}

	c := a.Clone()
	c.AndNot(b) // XXX: Should be in-place but isn't right now.
	d := b.Clone()
	d.AndNot(a)
	if c.GetCardinality() != 50 {
		t.Errorf("a-b Difference should have 50 bits set, but had %d", c.GetCardinality())
	}
	if d.GetCardinality() != 50 {
		t.Errorf("b-a Difference should have 150 bits set, but had %d", d.GetCardinality())
	}
	if c.Equals(d) {
		t.Errorf("Difference, here, should not be symmetric")
	}
}

func TestIterate(t *testing.T) {
	v := New(Size{Bits: 2, Chunks: 250}) // 500 bits.
	v.Add(0)
	v.Add(1)
	v.Add(2)
	buf := make([]uint32, 0, 10)
	j := uint32(0)
	for {
		var more bool
		buf, more = v.NextMany(j, buf, 1)
		if !more {
			break
		}
		j = buf[len(buf)-1] + 1
	}
	if buf[0] != 0 {
		t.Errorf("bug 0")
	}
	if buf[1] != 1 {
		t.Errorf("bug 1")
	}
	if buf[2] != 2 {
		t.Errorf("bug 2")
	}
	v.Add(10)
	v.Add(400)
	j = uint32(0)
	buf = buf[:0]
	for {
		var more bool
		buf, more = v.NextMany(j, buf, 1)
		if !more {
			break
		}
		j = buf[len(buf)-1] + 1
	}
	if buf[0] != 0 {
		t.Errorf("bug 0")
	}
	if buf[1] != 1 {
		t.Errorf("bug 1")
	}
	if buf[2] != 2 {
		t.Errorf("bug 2")
	}
	if buf[3] != 10 {
		t.Errorf("bug 3")
	}
	if buf[4] != 400 {
		t.Errorf("bug 4")
	}
}

func TestIterateSplit(t *testing.T) {
	v := New(Size{Bits: 2, Chunks: 250}) // 500 bits.
	bit := 0
	for i := 0; i < 250; i++ {
		v.Add(uint32(bit))
		bit += 2
	}
	buf := make([]uint32, 0, 10)
	j := uint32(0)
	for {
		var more bool
		buf, more = v.NextMany(j, buf, 3)
		if !more {
			break
		}
		j = buf[len(buf)-1] + 1
	}

	bit = 0
	for i := 0; i < 250; i++ {
		if int(buf[0]) != bit {
			t.Errorf("bug %d", bit)
		}
		buf = buf[1:]
		bit += 2
	}
}

func TestEachBatch(t *testing.T) {
	v := New(Size{Bits: 2, Chunks: 250}) // 500 bits.
	bit := 0
	data := []uint32{}
	for i := 0; i < 250; i++ {
		v.Add(uint32(bit))
		data = append(data, uint32(bit))
		bit += 2
	}
	v.EachBatch(1, func(batch []uint32) (bool, error) {
		if len(batch) != 1 {
			t.Errorf("wrong batch size")
		}
		if batch[0] != data[0] {
			t.Errorf("bug %d != %d", batch[0], data[0])
		}
		data = data[1:]
		return false, nil
	})
	if len(data) != 0 {
		t.Errorf("bug -- still have data")
	}
}

func TestRange(t *testing.T) {
	v := New(Size{Bits: 2, Chunks: 250}) // 500 bits.
	bit := 0
	data := []uint32{}
	for i := 0; i < 250; i++ {
		v.Add(uint32(bit))
		data = append(data, uint32(bit))
		bit += 2
	}
	i := 0
	for {
		batch := v.Range(i, i+1)
		i++
		if len(batch) == 0 {
			break
		}
		if batch[0] != data[0] {
			t.Fatalf("bug %d != %d", batch[0], data[0])
		}
		data = data[1:]
	}
	if len(data) != 0 {
		t.Errorf("bug -- still have data")
	}
}

/*
func TestFlipRange(t *testing.T) {
	b := New(sz)
	for _, v := range []uint32{1, 3, 5, 7, 9, 11, 13, 15} {
		b.Add(v)
	}
	b.FlipInt(4, 25)
	if b.GetCardinality() != 17 {
		t.Error("Unexpected value: ", b.GetCardinality())
		return
	}
	if !reflect.DeepEqual(b.ToArray(), []uint32{1, 3, 4, 6, 8, 10, 12, 14, 16, 17, 18, 19, 20, 21, 22, 23, 24}) {
		t.Error("Unexpected value: ", b.ToArray())
	}
	b.FlipInt(8, 24)
	if !reflect.DeepEqual(b.ToArray(), []uint32{1, 3, 4, 6, 9, 11, 13, 15, 24}) {
		t.Error("Unexpected value: ", b.ToArray())
		return
	}
}

func TestMarshalUnmarshalEmpty(t *testing.T) {
	b := New(sz)
	if !b.IsEmpty() {
		t.Error("bitmap should be empty")
		return
	}
	if b.GetCardinality() != 0 {
		t.Error("card should be 0")
		return
	}
	buf, err := b.Marshal()
	if err != nil {
		t.Error("Error marshalling: ", err)
		return
	}
	b1, err := NewBitmapFromBuf(buf, nbits, true)
	if err != nil {
		t.Error("Error marshalling: ", err)
		return
	}
	if !b1.Equals(b) {
		t.Error("bitmaps should be equal")
	}
	if !b1.IsEmpty() {
		t.Error("bitmap should be empty")
		return
	}
	if b1.GetCardinality() != 0 {
		t.Error("card should be 0")
		return
	}
}

func TestMarshalUnmarshalSmall(t *testing.T) {
	bits := []uint32{1, 3, 5, 7, 9, 11, 13, 15, 12345}
	b := New(sz)
	for _, v := range bits {
		b.Add(v)
	}
	buf, err := b.Marshal()
	if err != nil {
		t.Error("Error marshalling: ", err)
		return
	}
	b1, err := NewBitmapFromBuf(buf, nbits, true)
	if err != nil {
		t.Error("Error marshalling: ", err)
		return
	}
	if !b1.Equals(b) {
		t.Error("bitmaps should be equal")
	}
}

func TestMarshalUnmarshalBig(t *testing.T) {
	b := New(sz)
	for v := uint32(0); v < uint32(30000); v += 2 {
		b.Add(v)
	}
	buf, err := b.Marshal()
	if err != nil {
		t.Error("Error marshalling: ", err)
		return
	}
	b1, err := NewBitmapFromBuf(buf, nbits, true)
	if err != nil {
		t.Error("Error marshalling: ", err)
		return
	}
	if !b1.Equals(b) {
		t.Error("bitmaps should be equal")
	}
}

*/
