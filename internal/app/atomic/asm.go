package atomic

import (
	"crypto/sha256"
	"io"
)

// machine code output and symbols
type asm struct {
	instructions []uint32
	symbols      []symbol
	underscore   bool // exported symbols are underscored
}

type symbol struct {
	value  string
	offset int
	export bool
}

func (a *asm) addSymbol(value string, export bool) {
	if export && a.underscore {
		value = "_" + value
	}

	off := len(a.instructions) * 4
	if len(a.symbols) > 0 && a.symbols[len(a.symbols)-1].offset == off {
		return
	}
	a.symbols = append(a.symbols, symbol{
		value:  value,
		offset: off,
		export: export,
	})
}

func (a *asm) emit(i uint32) {
	a.instructions = append(a.instructions, i)
}

func (a *asm) align() {
	for len(a.instructions)&0x3 != 0 {
		a.emit(0)
	}
}

func (a *asm) writeAsm(w io.Writer) {
	b := []byte{0, 0, 0, 0}
	for _, v := range a.instructions {
		b[0] = byte(v)
		b[1] = byte(v >> 8)
		b[2] = byte(v >> 16)
		b[3] = byte(v >> 24)
		_, err := w.Write(b)
		if err != nil {
			shenanigans("Write failure on asm block.")
		}
	}
}

var boolToByteArray = map[bool][]byte{false: {0}, true: {1}}

func (a *asm) getHash() []byte {
	h := sha256.New()
	a.writeAsm(h)
	for _, s := range a.symbols {
		h.Write([]byte(s.value))
		h.Write([]byte{byte(s.offset), byte(s.offset >> 8), byte(s.offset >> 16), byte(s.offset >> 24)})
		h.Write(boolToByteArray[s.export])
	}
	return h.Sum(nil)
}

func (a *asm) size() int {
	return len(a.instructions) * 4
}

// returns the counts of local and exported symbols respectively
func (a *asm) symbolCounts() (int, int) {
	exported := 0
	for _, s := range a.symbols {
		if s.export {
			exported++
		}
	}
	return len(a.symbols) - exported, exported
}
