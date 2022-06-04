package atomic

import (
	"bytes"
	"encoding/binary"
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

type elf64header struct {
	magic1    uint64 // 0x00010102464c457f
	magic2    uint64 // 0
	fileType  uint16 // File type.
	machine   uint16 // Machine architecture.
	version   uint32 // ELF format version.
	entry     uint64 // Entry point.
	phoff     uint64 // Program header file offset.
	shoff     uint64 // Section header file offset.
	flags     uint32 // Architecture-specific flags.
	ehsize    uint16 // Size of ELF header in bytes.
	phentsize uint16 // Size of program header entry.
	phnum     uint16 // Number of program header entries.
	shentsize uint16 // Size of section header entry.
	shnum     uint16 // Number of section header entries.
	shstrndx  uint16 // Section name strings section.
}

type elf64section struct {
	name        uint32 // Section name, index in string tbl
	sectionType uint32 // Type of section
	flags       uint64 // Miscellaneous section attributes
	addr        uint64 // Section virtual addr at execution
	offset      uint64 // Section file offset
	size        uint64 // Size of section in bytes
	link        uint32 // Index of another section
	info        uint32 // Additional section information
	addralign   uint64 // Section alignment
	entsize     uint64 // Entry size if section holds table
}

const SIZEOF_ELF64HEADER = 64
const SIZEOF_ELF64SECTION = 64

func writeObjectFileElf(filename string, asms []asm) {
	file, err := os.Create(filename)
	defer file.Close()
	if err != nil {
		log.Fatal(err)
	}

	var buffer bytes.Buffer
	h := elf64header{
		magic1:    0x00010102464c457f,
		magic2:    0,
		fileType:  0x01, // object
		machine:   0xb7, // aarch64
		version:   0x01,
		entry:     0,
		phoff:     0,
		shoff:     SIZEOF_ELF64HEADER,
		flags:     0,
		ehsize:    SIZEOF_ELF64HEADER,
		phentsize: 0,
		phnum:     0,
		shentsize: SIZEOF_ELF64SECTION,
		shnum:     1,
		shstrndx:  1,
	}
	binary.Write(&buffer, binary.LittleEndian, h)

	stringTable, _ := buildStringTable(asms)

	s := elf64section{
		name:        0,
		sectionType: SHT_STRTAB,
		flags:       0,
		addr:        0,
		offset:      SIZEOF_ELF64HEADER + SIZEOF_ELF64SECTION,
		size:        uint64(len(stringTable)),
		link:        0,
		info:        0,
		addralign:   0,
		entsize:     uint64(len(stringTable)),
	}
	binary.Write(&buffer, binary.LittleEndian, s)

	buffer.Write(stringTable)

	// todo: the rest of the file - elf file support is not complete

	_, er := file.Write(buffer.Bytes())
	if er != nil {
		log.Fatal(err)
	}
}
