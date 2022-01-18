package boring

import (
	"math/bits"
)

// the wordSize of a bit set
const wordSize = 64

// log2WordSize is lg(wordSize)
const log2WordSize = 6

type bitmap struct {
	buf         []byte
	set         []uint64 // _ptr + 8 bytes
	cardinality int
}

func (b *bitmap) contains(v uint32) bool {
	return b.set[v>>log2WordSize]&(1<<(v&(wordSize-1))) != 0
}

func (b *bitmap) bitValue(v uint32) int {
	if int(b.set[v>>log2WordSize]&(1<<(v&(wordSize-1)))) != 0 {
		return 1
	}
	return 0
}

func (b *bitmap) add(v uint32) {
	idx := v >> log2WordSize
	previous := b.set[idx]
	mask := uint64(1 << (v & (wordSize - 1)))
	newb := previous | mask
	b.set[idx] = newb
	b.cardinality += int((previous ^ newb) >> (v & (wordSize - 1)))
}

func (b *bitmap) remove(v uint32) {
	if b.contains(v) {
		b.cardinality--
		b.set[v>>log2WordSize] &^= 1 << (v & (wordSize - 1))
	}
}

func (b *bitmap) and(o bitmap) {
	l := len(o.set)
	cnt := 0
	for i := 0; i < l; i++ {
		v := b.set[i] & o.set[i]
		cnt += bits.OnesCount64(v)
		b.set[i] = v
	}
	b.cardinality = int(cnt)
}

func (b *bitmap) or(o bitmap) {
	l := len(o.set)
	cnt := 0
	for i := 0; i < l; i++ {
		v := b.set[i] | o.set[i]
		cnt += bits.OnesCount64(v)
		b.set[i] = v
	}
	b.cardinality = int(cnt)
}

func (b *bitmap) andNot(o bitmap) {
	l := len(o.set)
	cnt := 0
	for i := 0; i < l; i++ {
		v := b.set[i] &^ o.set[i]
		cnt += bits.OnesCount64(v)
		b.set[i] = v
	}
	b.cardinality = int(cnt)
}

func (b *bitmap) andNotArray(o array) {
	for _, e := range o.content {
		b.remove(uint32(e))
	}
}

func (b *bitmap) flip(start, stop int) {
	startWord := start >> log2WordSize
	endWord := stop >> log2WordSize
	b.set[startWord] ^= ^(^uint64(0) << (start & (wordSize - 1)))
	for i := startWord; i < endWord; i++ {
		b.set[i] = ^b.set[i]
	}
	b.set[endWord] ^= ^uint64(0) >> (-stop & (wordSize - 1))
	b.cardinality = int(b.computeCardinality())
}

func (b *bitmap) computeCardinality() uint64 {
	cnt := 0
	for _, x := range b.set {
		cnt += bits.OnesCount64(x)
	}
	return uint64(cnt)
}

func (b *bitmap) equals(o bitmap) bool {
	l := len(o.set)
	for i := 0; i < l; i++ {
		if b.set[i] != o.set[i] {
			return false
		}
	}
	return true
}

func (b *bitmap) equalsArray(o array) bool {
	l := len(o.content)
	for i := 0; i < l; i++ {
		if !b.contains(uint32(o.content[i])) {
			return false
		}
	}
	return true
}

func (b *bitmap) nextSetMany16(buffer []uint16) {
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

func (b *bitmap) nextSetMany32(buffer []uint32) {
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
