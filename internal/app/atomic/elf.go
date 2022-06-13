package atomic

import (
	"bufio"
	"bytes"
	"log"
	"os"
)

// elf 64 object format for linux

const SHT_NULL = 0
const SHT_PROGBITS = 1
const SHT_SYMTAB = 2
const SHT_STRTAB = 3
const SHT_RELA = 4
const SHT_HASH = 5
const SHT_DYNAMIC = 6
const SHT_NOTE = 7
const SHT_NOBITS = 8
const SHT_REL = 9
const SHT_SHLIB = 10
const SHT_DYNSYM = 11
const SHT_NUM = 12
const SHT_LOPROC = 0x70000000
const SHT_HIPROC = 0x7fffffff
const SHT_LOUSER = 0x80000000
const SHT_HIUSER = 0xffffffff

const SHN_UNDEF = 0

const SHF_INFO_LINK = 0x40

const SHF_WRITE = 0x1
const SHF_ALLOC = 0x2
const SHF_EXECINSTR = 0x4
const SHF_RELA_LIVEPATCH = 0x00100000
const SHF_RO_AFTER_INIT = 0x00200000
const SHF_MASKPROC = 0xf0000000

const STB_LOCAL = 0
const STB_GLOBAL = 1
const STB_WEAK = 2

const STT_NOTYPE = 0
const STT_OBJECT = 1
const STT_FUNC = 2
const STT_SECTION = 3
const STT_FILE = 4
const STT_COMMON = 5
const STT_TLS = 6

type elf64header struct {
	magic1    uint64 // 0x00010102464c457f
	magic2    uint64 // 0
	fileType  uint16 // File type
	machine   uint16 // machine architecture
	version   uint32 // ELF format version
	entry     uint64 // entry point
	phoff     uint64 // program header file offset
	shoff     uint64 // section header file offset
	flags     uint32 // architecture-specific flags
	ehsize    uint16 // size of ELF header in bytes
	phentsize uint16 // size of program header entry
	phnum     uint16 // number of program header entries
	shentsize uint16 // size of section header entry
	shnum     uint16 // number of section header entries
	shstrndx  uint16 // section name strings section
}

type elf64section struct {
	name        uint32 // section name, index in string tbl
	sectionType uint32 // type of section
	flags       uint64 // miscellaneous section attributes
	addr        uint64 // section virtual addr at execution
	offset      uint64 // section file offset
	size        uint64 // size of section in bytes
	link        uint32 // index of another section
	info        uint32 // additional section information
	addralign   uint64 // section alignment
	entsize     uint64 // entry size if section holds table
}

type elf64symbol struct {
	name  uint32 // symbol name, index in string tbl
	info  uint8  // type and binding attributes
	other uint8  // no defined meaning, 0
	shndx uint16 // associated section index
	value uint64 // value of the symbol
	size  uint64 // associated symbol size
}

const SIZEOF_ELF64HEADER = 64
const SIZEOF_ELF64SECTION = 64
const SIZEOF_ELF64SYMBOL = 24

var exportSymbolTypesElf = map[bool]uint8{
	false: STB_LOCAL << 4,
	true:  (STB_GLOBAL << 4) | STT_FUNC,
}
var sectionStringTable, sectionStringIndexes = buildStringTable([]string{
	".text", ".symtab", ".strtab", ".shstrtab",
})

func writeObjectFileElf(filename string, asms []asm) {
	file, err := os.Create(filename)
	defer file.Close()
	if err != nil {
		log.Fatal(err)
	}

	buffer := bufio.NewWriter(file)

	stringTable, stringIndexes := buildAsmStringTable(asms)

	asmBlockSize := 0
	symbolCount := 0
	for _, a := range asms {
		asmBlockSize += a.size()
		symbolCount += len(a.symbols)
	}
	symbolBlockSize := symbolCount * SIZEOF_ELF64SYMBOL
	totalDataSize := asmBlockSize + symbolBlockSize + len(stringTable) + len(sectionStringTable)

	h := elf64header{
		magic1:    0x00010102464c457f,
		magic2:    0,
		fileType:  0x01, // object
		machine:   0xb7, // aarch64
		version:   0x01,
		entry:     0,
		phoff:     0,
		shoff:     uint64(SIZEOF_ELF64HEADER + totalDataSize),
		flags:     0,
		ehsize:    SIZEOF_ELF64HEADER,
		phentsize: 0,
		phnum:     0,
		shentsize: SIZEOF_ELF64SECTION,
		shnum:     5,
		shstrndx:  4,
	}
	writeStruct(buffer, h)

	// .text
	for _, a := range asms {
		a.writeAsm(buffer)
	}
	// .symtab
	for _, exp := range falseTrue {
		asmOffset := 0
		for _, a := range asms {
			for _, s := range a.symbols {
				if s.export == exp {
					sym := elf64symbol{
						name:  uint32(stringIndexes[s.value]),
						info:  exportSymbolTypesElf[s.export],
						other: 0,
						shndx: 1, // code in section 1
						value: uint64(asmOffset + s.offset),
						size:  0,
					}
					writeStruct(buffer, sym)
				}
			}
			asmOffset += a.size()
		}
	}
	// .strtab
	writeBytes(buffer, stringTable)
	// .shstrtab
	writeBytes(buffer, sectionStringTable)

	// section 0
	nullSection := elf64section{
		name:        0,
		sectionType: SHT_NULL,
		flags:       0,
		addr:        0,
		offset:      0,
		size:        0,
		link:        0,
		info:        0,
		addralign:   0,
		entsize:     0,
	}
	writeStruct(buffer, nullSection)

	// section 1
	asmSection := elf64section{
		name:        uint32(sectionStringIndexes[".text"]),
		sectionType: SHT_PROGBITS,
		flags:       SHF_ALLOC | SHF_EXECINSTR,
		addr:        0,
		offset:      SIZEOF_ELF64HEADER,
		size:        uint64(symbolCount * SIZEOF_ELF64SYMBOL),
		link:        0,
		info:        0,
		addralign:   8,
		entsize:     SIZEOF_ELF64SYMBOL,
	}
	writeStruct(buffer, asmSection)

	// section 2
	symSection := elf64section{
		name:        uint32(sectionStringIndexes[".symtab"]),
		sectionType: SHT_SYMTAB,
		flags:       0,
		addr:        0,
		offset:      uint64(SIZEOF_ELF64HEADER + asmBlockSize),
		size:        uint64(symbolCount * SIZEOF_ELF64SYMBOL),
		link:        3, // strtab in section 3
		info:        2,
		addralign:   8,
		entsize:     SIZEOF_ELF64SYMBOL,
	}
	writeStruct(buffer, symSection)

	// section 3
	strSection := elf64section{
		name:        uint32(sectionStringIndexes[".strtab"]),
		sectionType: SHT_STRTAB,
		flags:       0,
		addr:        0,
		offset:      uint64(SIZEOF_ELF64HEADER + asmBlockSize + symbolBlockSize),
		size:        uint64(len(stringTable)),
		link:        0,
		info:        0,
		addralign:   0,
		entsize:     0,
	}
	writeStruct(buffer, strSection)

	// section 4
	shStrSection := elf64section{
		name:        uint32(sectionStringIndexes[".shstrtab"]),
		sectionType: SHT_STRTAB,
		flags:       0,
		addr:        0,
		offset:      uint64(SIZEOF_ELF64HEADER + asmBlockSize + symbolBlockSize + len(stringTable)),
		size:        uint64(len(sectionStringTable)),
		link:        0,
		info:        0,
		addralign:   0,
		entsize:     0,
	}
	writeStruct(buffer, shStrSection)

	buffer.Flush()
}

func buildStringTable(values []string) ([]byte, map[string]int) {
	stringIndexes := make(map[string]int)
	var stringTable bytes.Buffer

	// no value at offset zero
	stringTable.WriteByte(0)
	// create the string table, with a map of string -> offset
	for _, v := range values {
		stringIndexes[v] = stringTable.Len()
		stringTable.WriteString(v)
		stringTable.WriteByte(0)
	}
	for stringTable.Len()&0x3 != 0 { // 32 bit pad
		stringTable.WriteByte(0)
	}
	return stringTable.Bytes(), stringIndexes
}
