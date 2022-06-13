package atomic

import (
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
)

type options struct {
	targetos    string
	outputdir   string
	verbose     bool
	concurrency int
	profile     string
}

func Atomic() {
	o := options{}

	a := args{}
	a.BoolArg('v', "verbose", "verbose output.", &o.verbose)
	a.StringArg('d', "outputdir", ".", false, "output directory.", nil, &o.outputdir)
	a.StringArg('t', "targetos", runtime.GOOS, false, "target OS.", []string{"darwin", "linux"}, &o.targetos)
	a.StringArg('p', "profile", "profile/arm64.profile", false, "cpu profile file.", nil, &o.profile)
	a.IntArg('c', "concurrency", "target concurrency thread count.", runtime.NumCPU(), &o.concurrency)
	tail := a.Process(os.Args, true, "atomic-source-files")

	for _, t := range tail {
		if !strings.HasSuffix(t, ".atomic") {
			a.FailWith("Source files must have file extension .atomic")
		}
	}

	profile := loadProfile(o, o.profile)
	compileFiles(profile, tail, o)
}

func compileFiles(profile profile, files []string, o options) {
	units := make([]unit, 0, len(files))
	unitChannel := make(chan unit, len(files))

	for fi, file := range files {
		go parseFile(o, file, unitChannel)

		for fi-len(units) > o.concurrency {
			units = append(units, <-unitChannel)
		}
	}
	for len(units) < len(files) {
		units = append(units, <-unitChannel)
	}

	// create reference structure from all files
	r := reference{
		structs:   map[string]*structNode{},
		functions: map[string]*functionNode{},
	}
	for _, u := range units {
		r.populate(&u.ast)
	}

	// compile each file
	objectChannel := make(chan string, len(units))
	for _, u := range units {
		// collect functions for each file
		functions := collectFunctions(&u.ast)
		asms := make([]asm, 0, len(functions))
		asmChannel := make(chan asm, len(asms))

		for fi, fa := range functions {
			go compileFunction(fa, &profile, &r, asmChannel)

			for fi-len(asms) > o.concurrency {
				asms = append(asms, <-asmChannel)
			}
		}
		for len(asms) < len(functions) {
			asms = append(asms, <-asmChannel)
		}
		// sort reordered results by function name
		sort.Slice(asms, func(i, j int) bool {
			return asms[i].symbols[0].value < asms[j].symbols[0].value
		})

		if o.verbose {
			for _, a := range asms {
				fmt.Printf("ASM %x %s\n", a.getHash(), a.symbols[0].value)
			}
		}

		go writeObject(asms, objectFileName(o.outputdir, u.filename), o.targetos, objectChannel)
	}
	for range units {
		<-objectChannel
	}

	if o.verbose {
		// sort reordered results by file name
		sort.Slice(units, func(i, j int) bool {
			return units[i].filename < units[j].filename
		})

		for _, u := range units {
			fmt.Printf("AST %x %s\n", u.ast.getHash(), u.filename)
		}
	}
}

func writeObject(asms []asm, filename string, targetos string, objectChannel chan string) {
	switch targetos {
	case "darwin":
		writeObjectFileMach(filename, asms)
	case "linux":
		writeObjectFileElf(filename, asms)
	default:
		shenanigans("Object format not supported for %s", targetos)
	}

	objectChannel <- filename
}

func compileFunction(fa ast, profile *profile, r *reference, asmChannel chan asm) {
	asm := asm{
		instructions: make([]uint32, 0, 128),
		symbols:      make([]symbol, 0, 16),
		underscore:   profile.targetos == "darwin",
	}

	dc := fa.emit(newFrame(r), profile, &asm)
	if dc != nil {
		dc()
	}
	asm.align()

	asmChannel <- asm
}

func objectFileName(dir string, source string) string {
	source = strings.TrimSuffix(source, ".atomic")
	dirs := strings.Split(source, "/")                // remove dirs
	source = strings.Split(dirs[len(dirs)-1], ".")[0] // remove extensions
	return fmt.Sprintf("%s/%s.o", dir, source)
}

func shenanigans(format string, a ...interface{}) {
	fmt.Println()
	fmt.Println("Atomic Shenanigans:")
	fmt.Printf(format, a...)
	fmt.Println()
	fmt.Println()
	os.Exit(1)
}
