package boring

type array struct {
	buf     []byte
	content []uint16 // _ptr + 8 bytes
	sz      int
}

func (b *array) add(v uint32) {
	x := uint16(v)
	// Special case adding to the end of the container.
	l := len(b.content)
	if l > 0 && l < b.sz && b.content[l-1] < x {
		b.content = append(b.content, x)
		return
	}

	loc := binarySearch(b.content, x)
	if loc < 0 {
		s := b.content
		i := -loc - 1
		s = append(s, 0)
		copy(s[i+1:], s[i:])
		s[i] = x
		b.content = s
		return
	}
}

func (b *array) contains(v uint32) bool {
	x := uint16(v)
	return binarySearch(b.content, x) >= 0
}

func (b *array) remove(v uint32) {
	x := uint16(v)
	loc := binarySearch(b.content, x)
	if loc >= 0 {
		s := b.content
		s = append(s[:loc], s[loc+1:]...)
		b.content = s
	}
}

func (b *array) and(o array) {
	length := intersection2by2(b.content, o.content, b.content)
	b.content = b.content[:length]
}

func (b *array) andBitmap(o bitmap) {
	content := make([]uint16, len(b.content))
	pos := 0
	for _, v := range b.content {
		content[pos] = v
		pos += o.bitValue(uint32(v))
	}
	content = content[:pos]
	b.content = toUint16Slice(b.buf[8:], len(content))
	copy(b.content, content)
}

func (b *array) or(o array) {
	lb := len(b.content)
	lo := len(o.content)
	max := lb + lo

	// Since we never utilitize more than 50% of the buffer space
	// we know the max size is never more than the entire buffer.
	b.content = toUint16Slice(b.buf[8:], max)
	copy(b.content[lo:max], o.content[:lb])
	l := union2by2(b.content[lb:max], o.content, b.content)
	b.content = b.content[:l]
}

func (b *array) andNot(o array) {
	length := difference(b.content, o.content, b.content)
	b.content = b.content[:length]
}

func (b *array) andNotBitmap(o bitmap) {
	src := make([]uint16, len(b.content))
	copy(src, b.content)

	b.content = b.content[:0]
	for _, e := range b.content {
		v := uint32(e)
		if !o.contains(v) {
			b.add(v)
		}
	}
}

func (b *array) equals(o array) bool {
	l := len(b.content)
	for i := 0; i < l; i++ {
		if b.content[i] != o.content[i] {
			return false
		}
	}
	return true
}

func (b *array) equalsBitmap(o bitmap) bool {
	l := len(b.content)
	for i := 0; i < l; i++ {
		if !o.contains(uint32(b.content[i])) {
			return false
		}
	}
	return true
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func binarySearch(array []uint16, ikey uint16) int {
	low := 0
	high := len(array) - 1
	for low+16 <= high {
		middleIndex := int(uint32(low+high) >> 1)
		middleValue := array[middleIndex]
		if middleValue < ikey {
			low = middleIndex + 1
		} else if middleValue > ikey {
			high = middleIndex - 1
		} else {
			return middleIndex
		}
	}
	for ; low <= high; low++ {
		val := array[low]
		if val >= ikey {
			if val == ikey {
				return low
			}
			break
		}
	}
	return -(low + 1)
}

func union2by2(set1 []uint16, set2 []uint16, buffer []uint16) int {
	pos := 0
	k1 := 0
	k2 := 0
	if 0 == len(set2) {
		buffer = buffer[:len(set1)]
		copy(buffer, set1[:])
		return len(set1)
	}
	if 0 == len(set1) {
		buffer = buffer[:len(set2)]
		copy(buffer, set2[:])
		return len(set2)
	}
	s1 := set1[k1]
	s2 := set2[k2]
	buffer = buffer[:cap(buffer)]
	for {
		if s1 < s2 {
			buffer[pos] = s1
			pos++
			k1++
			if k1 >= len(set1) {
				copy(buffer[pos:], set2[k2:])
				pos += len(set2) - k2
				break
			}
			s1 = set1[k1]
		} else if s1 == s2 {
			buffer[pos] = s1
			pos++
			k1++
			k2++
			if k1 >= len(set1) {
				copy(buffer[pos:], set2[k2:])
				pos += len(set2) - k2
				break
			}
			if k2 >= len(set2) {
				copy(buffer[pos:], set1[k1:])
				pos += len(set1) - k1
				break
			}
			s1 = set1[k1]
			s2 = set2[k2]
		} else { // if (set1[k1]>set2[k2])
			buffer[pos] = s2
			pos++
			k2++
			if k2 >= len(set2) {
				copy(buffer[pos:], set1[k1:])
				pos += len(set1) - k1
				break
			}
			s2 = set2[k2]
		}
	}
	return pos
}

func intersection2by2(
	set1 []uint16,
	set2 []uint16,
	buffer []uint16) int {

	if len(set1)*64 < len(set2) {
		return onesidedgallopingintersect2by2(set1, set2, buffer)
	} else if len(set2)*64 < len(set1) {
		return onesidedgallopingintersect2by2(set2, set1, buffer)
	} else {
		return localintersect2by2(set1, set2, buffer)
	}
}

func onesidedgallopingintersect2by2(
	smallset []uint16,
	largeset []uint16,
	buffer []uint16) int {

	if 0 == len(smallset) {
		return 0
	}
	buffer = buffer[:cap(buffer)]
	k1 := 0
	k2 := 0
	pos := 0
	s1 := largeset[k1]
	s2 := smallset[k2]
mainwhile:

	for {
		if s1 < s2 {
			k1 = advanceUntil(largeset, k1, len(largeset), s2)
			if k1 == len(largeset) {
				break mainwhile
			}
			s1 = largeset[k1]
		}
		if s2 < s1 {
			k2++
			if k2 == len(smallset) {
				break mainwhile
			}
			s2 = smallset[k2]
		} else {

			buffer[pos] = s2
			pos++
			k2++
			if k2 == len(smallset) {
				break
			}
			s2 = smallset[k2]
			k1 = advanceUntil(largeset, k1, len(largeset), s2)
			if k1 == len(largeset) {
				break mainwhile
			}
			s1 = largeset[k1]
		}

	}
	return pos
}

func localintersect2by2(
	set1 []uint16,
	set2 []uint16,
	buffer []uint16) int {

	if (0 == len(set1)) || (0 == len(set2)) {
		return 0
	}
	k1 := 0
	k2 := 0
	pos := 0
	buffer = buffer[:cap(buffer)]
	s1 := set1[k1]
	s2 := set2[k2]
mainwhile:
	for {
		if s2 < s1 {
			for {
				k2++
				if k2 == len(set2) {
					break mainwhile
				}
				s2 = set2[k2]
				if s2 >= s1 {
					break
				}
			}
		}
		if s1 < s2 {
			for {
				k1++
				if k1 == len(set1) {
					break mainwhile
				}
				s1 = set1[k1]
				if s1 >= s2 {
					break
				}
			}

		} else {
			// (set2[k2] == set1[k1])
			buffer[pos] = s1
			pos++
			k1++
			if k1 == len(set1) {
				break
			}
			s1 = set1[k1]
			k2++
			if k2 == len(set2) {
				break
			}
			s2 = set2[k2]
		}
	}
	return pos
}

func advanceUntil(
	array []uint16,
	pos int,
	length int,
	min uint16) int {
	lower := pos + 1

	if lower >= length || array[lower] >= min {
		return lower
	}

	spansize := 1

	for lower+spansize < length && array[lower+spansize] < min {
		spansize *= 2
	}
	var upper int
	if lower+spansize < length {
		upper = lower + spansize
	} else {
		upper = length - 1
	}

	if array[upper] == min {
		return upper
	}

	if array[upper] < min {
		// means array has no item >= min
		// pos = array.length;
		return length
	}

	// we know that the next-smallest span was too small
	lower += (spansize >> 1)

	mid := 0
	for lower+1 != upper {
		mid = (lower + upper) >> 1
		if array[mid] == min {
			return mid
		} else if array[mid] < min {
			lower = mid
		} else {
			upper = mid
		}
	}
	return upper

}

func difference(set1 []uint16, set2 []uint16, buffer []uint16) int {
	if 0 == len(set2) {
		buffer = buffer[:len(set1)]
		for k := 0; k < len(set1); k++ {
			buffer[k] = set1[k]
		}
		return len(set1)
	}
	if 0 == len(set1) {
		return 0
	}
	pos := 0
	k1 := 0
	k2 := 0
	buffer = buffer[:cap(buffer)]
	s1 := set1[k1]
	s2 := set2[k2]
	for {
		if s1 < s2 {
			buffer[pos] = s1
			pos++
			k1++
			if k1 >= len(set1) {
				break
			}
			s1 = set1[k1]
		} else if s1 == s2 {
			k1++
			k2++
			if k1 >= len(set1) {
				break
			}
			s1 = set1[k1]
			if k2 >= len(set2) {
				for ; k1 < len(set1); k1++ {
					buffer[pos] = set1[k1]
					pos++
				}
				break
			}
			s2 = set2[k2]
		} else { // if (val1>val2)
			k2++
			if k2 >= len(set2) {
				for ; k1 < len(set1); k1++ {
					buffer[pos] = set1[k1]
					pos++
				}
				break
			}
			s2 = set2[k2]
		}
	}
	return pos

}
