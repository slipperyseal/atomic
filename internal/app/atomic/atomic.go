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
	a.IntArg('c', "concurrency", "target concurrency cpu count.", runtime.NumCPU(), &o.concurrency)
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
	r := reference{
		structs:   map[string]*structNode{},
		functions: map[string]*functionNode{},
	}

	// parse all files and combine into reference
	rootAsts := make([]ast, 0, len(files))
	astChannel := make(chan ast, len(files))

	for fi, file := range files {
		go parseFile(o, file, astChannel)

		for fi-len(rootAsts) > o.concurrency { // if backlog greater than concurrency wait on channel
			rootAst := <-astChannel
			rootAsts = append(rootAsts, rootAst)
			r.populate(&rootAst)
		}
	}
	for len(rootAsts) < len(files) {
		rootAst := <-astChannel
		rootAsts = append(rootAsts, rootAst)
		r.populate(&rootAst)
	}

	// compile each file
	for i, rootAst := range rootAsts {
		file := files[i]

		// collect functions for each file
		functions := make([]ast, 0, 10)
		for _, a := range rootAst.sub {
			switch a.node.(type) {
			case *functionNode:
				functions = append(functions, a)
			}
		}

		asms := make([]asm, 0, len(functions))
		asmChannel := make(chan asm, len(asms))

		for fi, fa := range functions {
			go compileFunction(fa, &profile, asmChannel, &r)

			for fi-len(asms) > o.concurrency { // if backlog greater than concurrency wait on channel
				asms = append(asms, <-asmChannel)
			}
		}
		for len(asms) < len(functions) {
			asms = append(asms, <-asmChannel)
		}

		sort.Slice(asms, func(i, j int) bool {
			return asms[i].symbols[0].value < asms[j].symbols[0].value
		})

		filename := objectFileName(o.outputdir, file)

		switch o.targetos {
		case "darwin":
			writeObjectFileMach(filename, asms)
		case "linux":
			writeObjectFileElf(filename, asms)
		default:
			shenanigans("Object format not supported for %s", o.targetos)
		}
	}
}

func compileFunction(fa ast, profile *profile, asmChannel chan asm, r *reference) {
	asm := asm{
		instructions: make([]uint32, 0, 128),
		symbols:      make([]symbol, 0, 16),
	}

	dc := fa.emit(newFrame(r), profile, &asm)
	if dc != nil {
		dc()
	}

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
