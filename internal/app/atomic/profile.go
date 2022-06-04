package atomic

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
)

type register struct {
	index   int
	name    string
	scratch bool
	param   bool
	result  bool
	link    bool
}

type instruction struct {
	name   string // instruction name, not unique
	bits   uint32 // bits to set
	mask   uint32 // mask to extract bits
	params []param
	order  []string
}

type param struct {
	code   rune
	offset int
	len    int
}

type profile struct {
	instructions []instruction
	registers    []register
}

func (i instruction) isSupported() bool {
	for _, o := range i.order {
		// ignoring and SIMD, Vector or Float instructions for now
		if len(o) == 2 && (strings.HasPrefix(o, "S") || strings.HasPrefix(o, "V") || strings.HasPrefix(o, "F")) {
			return false
		}
		if strings.HasPrefix(o, "SIMD") {
			return false
		}
	}
	return true
}

func loadProfile(o options, filename string) profile {
	file, err := os.Open(filename)
	if err != nil {
		shenanigans("Failed to open file: %s %v", filename, err)
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	dupe := make(map[uint32]instruction)
	instructions := []instruction{}
	registers := []register{}

	lnum := 0
	for {
		bytes, _, err := reader.ReadLine()
		if err == io.EOF {
			break
		} else if err != nil {
			shenanigans("Error reading source %v", err)
		}

		lnum++
		line := strings.Split(strings.TrimSpace(string(bytes)), "#")[0] // remove comments

		if len(line) == 0 {
			continue
		}

		if strings.HasPrefix(line, "!") { // register
			r := parseRegister(line, lnum, o.targetos)
			if r.name == "" {
				continue
			}
			registers = append(registers, r)
			if o.verbose {
				fmt.Printf("%+v\n", r)
			}
			continue
		}

		ins, template := parseInstruction(line, lnum)

		if !ins.isSupported() {
			continue
		}

		d, exists := dupe[ins.bits]
		if exists {
			if o.verbose {
				fmt.Printf("Instruction with duplicate bits.\nIgnoring: %+v\nEquals:   %+v\n", ins, d)
			}
			continue
		}

		instructions = append(instructions, ins)
		dupe[ins.bits] = ins

		if o.verbose {
			fmt.Printf("%s %+v mask bits: %08x : %08x\n", template, ins, ins.mask, ins.bits)
		}
	}

	return profile{
		instructions: instructions,
		registers:    registers,
	}
}

func parseRegister(line string, lnum int, targetOs string) register {
	split := strings.Split(line, "!")
	reg := strings.TrimSpace(split[1])
	segments := strings.Split(reg, " ")
	tags := segments[2:]

	if contains(tags, "no"+targetOs) {
		return register{}
	}

	num, err := strconv.Atoi(segments[0])
	if err != nil {
		shenanigans("Unable to parse register index on line %d\n", lnum)
	}
	r := register{
		index:   num,
		name:    segments[1],
		scratch: contains(tags, "scratch"),
		param:   contains(tags, "param"),
		result:  contains(tags, "result"),
		link:    contains(tags, "link"),
	}
	return r
}

func parseInstruction(line string, lnum int) (instruction, string) {
	split := strings.Split(line, "-")
	template := strings.Replace(split[0], " ", "", -1)
	asm := strings.TrimSpace(split[1])
	segments := strings.Split(asm, " ")

	if len(template) != 32 {
		shenanigans("Instruction template length not 32 on line %d\n", lnum)
	}

	paramMap := make(map[rune]param)
	var bits uint32 = 0
	var mask uint32 = 0
	for pos, char := range template {
		i := 31 - pos
		if char == '1' {
			bits |= 1 << i
		}
		if char == '0' || char == '1' {
			mask |= 1 << i
		} else if char != 'x' {
			p, x := paramMap[char]
			if !x {
				p := param{
					code:   char,
					offset: i,
					len:    1,
				}
				paramMap[char] = p
			} else {
				p.len++
				p.offset = i
				paramMap[char] = p
			}
		}
	}

	params := make([]param, 0, len(paramMap))
	for _, value := range paramMap {
		params = append(params, value)
	}
	sort.Slice(params, func(i, j int) bool {
		return params[i].code < params[j].code
	})

	ins := instruction{
		name:   segments[0],
		bits:   bits,
		mask:   mask,
		params: params,
		order:  segments[1:],
	}
	return ins, template
}

// reverse match of instruction binary (set regs to zero) eg. 8b000000 = add x0, x0, x0
func (p *profile) match(bin uint32) {
	fmt.Printf("Matching %08x\n", bin)
	for i := 31; i >= 0; i-- {
		fmt.Print((bin >> i) & 1)
		if i&3 == 0 {
			fmt.Print(" ")
		}
	}
	fmt.Println()

	for _, ins := range p.instructions {
		mb := bin & ins.mask
		if mb == ins.bits {
			fmt.Println(ins)
		}
	}
}

func (p *profile) findNoargs(name string) uint32 {
	for _, ins := range p.instructions {
		if ins.name == name && len(ins.params) == 0 {
			return ins.bits
		}
	}
	shenanigans("Couldnt find instruction %s\n", name)
	return 0
}

func (p *profile) find(name string, paramSet string) instruction {
	for _, ins := range p.instructions {
		if ins.name == name && ins.hasParams(paramSet) {
			return ins
		}
	}
	shenanigans("Couldnt find instruction %s with %s\n", name, paramSet)
	return instruction{}
}

func (i instruction) set(paramSet string, params ...int) uint32 {
	b := i.bits
	for _, ps := range i.params {
		for i, p := range paramSet {
			if ps.code == p {
				b |= uint32(params[i] << ps.offset)
			}
		}
	}
	return b
}

func (i *instruction) hasParams(codes string) bool {
	if len(i.params) != len(codes) {
		return false
	}
	matches := 0
	for _, c := range codes {
		for _, p := range i.params {
			if c == p.code {
				matches++
			}
		}
	}
	return len(codes) == matches
}

func (p *profile) findReg(name string, n int) uint32 {
	for _, ins := range p.instructions {
		if ins.name == name {
			b := ins.bits
			for _, ps := range ins.params {
				if ps.code == 'n' {
					b |= uint32(n << ps.offset)
					return b
				}
			}
		}
	}
	shenanigans("Couldn't find instruction %s\n", name)
	return 0
}

func (p *profile) findLinkRegister() register {
	for _, r := range p.registers {
		if r.link {
			return r
		}
	}
	shenanigans("Couldn't find link register\n")
	return register{}
}

func contains(sa []string, s string) bool {
	for _, v := range sa {
		if v == s {
			return true
		}
	}
	return false
}
