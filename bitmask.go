package ecs

type Bitmask struct {
	bits []uint64
}

func NewBitmask(bits ...uint) Bitmask {
	var bm Bitmask
	for _, bit := range bits {
		bm.Set(bit)
	}

	return bm
}

// Set sets the i-th bit
func (b *Bitmask) Set(i uint) {
	word, bit := i/64, i%64

	if word >= uint(len(b.bits)) {
		b.bits = append(b.bits, make([]uint64, word-uint(len(b.bits))+1)...)
	}

	b.bits[word] |= 1 << bit
}

// Unset unsets the i-th bit
func (b *Bitmask) Unset(i uint) {
	word, bit := i/64, i%64

	if word < uint(len(b.bits)) {
		b.bits[word] &^= 1 << bit
	}
}

// HasAll returns true if all bits are set
func (b *Bitmask) HasAll(bits Bitmask) bool {
	for i, word := range bits.bits {
		if word == 0 {
			continue
		}
		if i >= len(b.bits) {
			return false
		}
		if (b.bits[i] & word) != word {
			return false
		}
	}
	return true
}

// Has returns true if the i-th bit is set
func (b *Bitmask) Has(i uint) bool {
	word, bit := i/64, i%64
	return word < uint(len(b.bits)) && b.bits[word]&(1<<bit) != 0
}

// Equals returns true if both bitmasks have the same bits set
func (b *Bitmask) Equals(other Bitmask) bool {
	maxLen := max(len(other.bits), len(b.bits))

	for i := range maxLen {
		var bWord, otherWord uint64
		if i < len(b.bits) {
			bWord = b.bits[i]
		}
		if i < len(other.bits) {
			otherWord = other.bits[i]
		}
		if bWord != otherWord {
			return false
		}
	}

	return true
}
