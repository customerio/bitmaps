package boring

import (
	"reflect"
	"testing"
)

var nbits = 30000

func TestConvert(t *testing.T) {
	b := NewBitmap(nbits)
	for v := uint32(0); v < uint32(30000); v += 2 {
		b.Add(v)
	}
}

func TestClone(t *testing.T) {
	b := NewBitmap(nbits)
	for v := uint32(0); v < uint32(30000); v += 2 {
		b.Add(v)
	}
	b1 := NewBitmap(nbits)
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
	b := NewBitmap(nbits)
	for v := uint32(0); v < uint32(30000); v += 100 {
		o := NewBitmap(nbits)
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
	a := NewBitmap(nbits)
	c := NewBitmap(nbits)
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
	a := NewBitmap(nbits)
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
	a := NewBitmap(nbits)
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
	a := NewBitmap(nbits)
	b := NewBitmap(nbits)
	for i := uint32(1); i < 100; i += 2 {
		a.Add(i)
		b.Add(i - 1)
	}
	for i := uint32(100); i < 200; i++ {
		b.Add(i)
	}
	c := OrBitmaps(a, b)
	d := OrBitmaps(b, a)
	if c.GetCardinality() != 200 {
		t.Errorf("Union should have 200 bits set, but had %d", c.GetCardinality())
	}
	if !c.Equals(d) {
		t.Errorf("Union should be symmetric")
	}
}

func TestOr(t *testing.T) {
	a := NewBitmap(nbits)
	b := NewBitmap(nbits)
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
	a := NewBitmap(nbits)
	b := NewBitmap(nbits)
	for i := uint32(1); i < 100; i += 2 {
		a.Add(i)
		b.Add(i - 1)
		b.Add(i)
	}
	for i := uint32(100); i < 200; i++ {
		b.Add(i)
	}
	c := AndBitmaps(a, b)
	d := AndBitmaps(b, a)
	if c.GetCardinality() != 50 {
		t.Errorf("Intersection should have 50 bits set, but had %d", c.GetCardinality())
	}
	if !c.Equals(d) {
		t.Errorf("Intersection should be symmetric")
	}
}

func TestAnd(t *testing.T) {
	a := NewBitmap(nbits)
	b := NewBitmap(nbits)
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
	a := NewBitmap(nbits)
	b := NewBitmap(nbits)
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
	a := NewBitmap(nbits)
	b := NewBitmap(nbits)
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

func TestFlipRange(t *testing.T) {
	b := NewBitmap(nbits)
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
	b := NewBitmap(nbits)
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
	b := NewBitmap(nbits)
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
	b := NewBitmap(nbits)
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
