package atomic

import (
	"bytes"
	"encoding/binary"
	"log"
	"os"
)

// mach-o 64 structures for mac os darwin

type mach64header struct {
	magicNumber          uint32
	cpuType              uint32
	cpuSubType           uint32
	fileType             uint32
	numberOfLoadCommands uint32
	sizeOfLoadCommands   uint32
	flags                uint32
	reserved             uint32
}

type lcSegment64 struct {
	command             uint32
	commandSize         uint32
	segmentName         [16]uint8
	vmAddress           uint64
	vmSize              uint64
	fileOffset          uint64
	fileSize            uint64
	maxVmProtection     uint32
	initialVmProtection uint32
	numberOfSections    uint32
	flags               uint32
}

type section64 struct {
	sectionName         [16]uint8
	segmentName         [16]uint8
	address             uint64
	size                uint64
	offset              uint32
	alignment           uint32
	relocationsOffset   uint32
	numberOfRelocations uint32
	flags               uint32
	reserved1           uint32
	reserved2           uint32
	reserved3           uint32
}

type lsBuildVersion struct {
	command     uint32
	commandSize uint32
	platform    uint32
	minos       uint32
	sdk         uint32
	ntools      uint32
}

type lcSymTab struct {
	command           uint32
	commandSize       uint32
	symbolTableOffset uint32
	numberOfSymbols   uint32
	stringTableOffset uint32
	stringTableSize   uint32
}

type lcDySymTab struct {
	command                 uint32
	commandSize             uint32
	locSymbolIndex          uint32
	locSymbolNumber         uint32
	definedExtSymbolndex    uint32
	definedExtSymboNumber   uint32
	undefinedExtSymbolndex  uint32
	undefinedExtSymboNumber uint32
	tocOffset               uint32
	tocEntries              uint32
	moduleTableOffset       uint32
	moduleTableEntries      uint32
	extRefTableOffset       uint32
	extRefTableEntries      uint32
	indSymTableOffset       uint32
	indSymTableEntries      uint32
	extRelocTableOffset     uint32
	extRelocTableEntries    uint32
	locRelocTableOffset     uint32
	locRelocTableEntries    uint32
}

type machSymbol struct {
	stringTableIndex uint32
	symbolType       uint8
	sectionIndex     uint8
	description      uint16
	value            uint64
}

const SIZEOF_MACH64HEADER = 32
const SIZEOF_LCSEGMENT64 = 72
const SIZEOF_SECTION64 = 80
const SIZEOF_LSBUILDVERSION = 24
const SIZEOF_LCSYMTAB = 24
const SIZEOF_LCDYSYMTAB = 80
const SIZEOF_MACHSYMBOL = 16

var falseTrue = []bool{false, true}
var exportSymbolTypes = map[bool]uint8{false: 0xe, true: 0xf}

func writeObjectFileMach(filename string, asms []asm) {
	file, err := os.Create(filename)
	defer file.Close()
	if err != nil {
		log.Fatal(err)
	}

	// align assembler and calculate combined lengths
	asmSize := 0
	localSymbols := 0
	exportedSymbols := 0
	for i, _ := range asms {
		a := &asms[i]
		a.align()
		asmSize += a.size()
		loc, exp := a.symbolCounts()
		localSymbols += loc
		exportedSymbols += exp
	}

	loadSize := SIZEOF_LCSEGMENT64 + SIZEOF_SECTION64 + SIZEOF_LSBUILDVERSION + SIZEOF_LCSYMTAB + SIZEOF_LCDYSYMTAB
	asmOff := SIZEOF_MACH64HEADER + loadSize
	symOff := asmOff + asmSize

	stringTable, stringIndexes := buildStringTable(asms)

	stringOff := symOff + (SIZEOF_MACHSYMBOL * len(stringIndexes))

	var buffer bytes.Buffer
	h := mach64header{
		magicNumber:          0xfeedfacf,
		cpuType:              0x0100000c,
		cpuSubType:           0,
		fileType:             1,
		numberOfLoadCommands: 4,
		sizeOfLoadCommands:   uint32(loadSize),
		flags:                0,
		reserved:             0,
	}
	binary.Write(&buffer, binary.LittleEndian, h)

	lc64 := lcSegment64{
		command:             0x00000019,
		commandSize:         SIZEOF_LCSEGMENT64 + SIZEOF_SECTION64,
		segmentName:         [16]uint8{},
		vmAddress:           0,
		vmSize:              uint64(asmSize),
		fileOffset:          uint64(asmOff),
		fileSize:            uint64(asmSize),
		maxVmProtection:     0x7,
		initialVmProtection: 0x7,
		numberOfSections:    1,
		flags:               0,
	}
	binary.Write(&buffer, binary.LittleEndian, lc64)

	text := section64{
		sectionName:         fixedString16("__text"),
		segmentName:         fixedString16("__TEXT"),
		address:             0,
		size:                uint64(asmSize),
		offset:              uint32(asmOff),
		alignment:           4,
		relocationsOffset:   0,
		numberOfRelocations: 0,
		flags:               0x80000400,
		reserved1:           0,
		reserved2:           0,
		reserved3:           0,
	}
	binary.Write(&buffer, binary.LittleEndian, text)

	build := lsBuildVersion{
		command:     0x32,
		commandSize: SIZEOF_LSBUILDVERSION,
		platform:    0x1,
		minos:       0xc0000,
		sdk:         0,
		ntools:      0,
	}
	binary.Write(&buffer, binary.LittleEndian, build)

	symTab := lcSymTab{
		command:           0x00000002,
		commandSize:       SIZEOF_LCSYMTAB,
		symbolTableOffset: uint32(symOff),
		numberOfSymbols:   uint32(len(stringIndexes)),
		stringTableOffset: uint32(stringOff),
		stringTableSize:   uint32(len(stringTable)),
	}
	binary.Write(&buffer, binary.LittleEndian, symTab)

	dySymTab := lcDySymTab{
		command:                 0x0000000b,
		commandSize:             SIZEOF_LCDYSYMTAB,
		locSymbolIndex:          0,
		locSymbolNumber:         uint32(localSymbols),
		definedExtSymbolndex:    uint32(localSymbols),
		definedExtSymboNumber:   uint32(exportedSymbols),
		undefinedExtSymbolndex:  uint32(localSymbols + exportedSymbols),
		undefinedExtSymboNumber: 0,
		tocOffset:               0,
		tocEntries:              0,
		moduleTableOffset:       0,
		moduleTableEntries:      0,
		extRefTableOffset:       0,
		extRefTableEntries:      0,
		indSymTableOffset:       0,
		indSymTableEntries:      0,
		extRelocTableOffset:     0,
		extRelocTableEntries:    0,
		locRelocTableOffset:     0,
		locRelocTableEntries:    0,
	}
	binary.Write(&buffer, binary.LittleEndian, dySymTab)

	for _, a := range asms {
		a.writeAsm(&buffer)
	}

	// local symbols first, then exported
	for _, exp := range falseTrue {
		asmOffset := 0
		for _, a := range asms {
			for _, s := range a.symbols {
				if s.export == exp {
					ms := machSymbol{
						stringTableIndex: uint32(stringIndexes[s.value]),
						symbolType:       exportSymbolTypes[s.export],
						sectionIndex:     1,
						description:      0,
						value:            uint64(asmOffset + s.offset),
					}
					binary.Write(&buffer, binary.LittleEndian, ms)
				}
			}
			asmOffset += a.size()
		}
	}

	buffer.Write(stringTable)

	_, er := file.Write(buffer.Bytes())
	if er != nil {
		log.Fatal(err)
	}
}

func fixedString16(value string) [16]uint8 {
	slice := [16]uint8{}
	for i, v := range value {
		slice[i] = uint8(v)
	}
	return slice
}

func buildStringTable(asms []asm) ([]byte, map[string]int) {
	stringIndexes := make(map[string]int)
	var stringTable bytes.Buffer

	// no value at offset zero
	stringTable.WriteByte(0)
	// create the string table, with a map of string -> offset
	for _, a := range asms {
		for _, s := range a.symbols {
			stringIndexes[s.value] = stringTable.Len()
			stringTable.WriteString(s.value)
			stringTable.WriteByte(0)
		}
	}
	for stringTable.Len()&0x3 != 0 { // 32 bit pad
		stringTable.WriteByte(0)
	}
	return stringTable.Bytes(), stringIndexes
}
