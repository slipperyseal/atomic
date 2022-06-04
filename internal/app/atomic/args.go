package atomic

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type args struct {
	argSlice  []arg
	program   string
	tailLabel string
}

type arg struct {
	letter       byte
	name         string
	required     bool
	dephault     string
	description  string
	choices      []string
	outputString *string
	outputBool   *bool
	outputInt    *int
	resolved     bool
}

func (a *args) Process(osargs []string, requireTail bool, tailLabel string) []string {
	ss := strings.Split(osargs[0], "/")
	a.program = ss[len(ss)-1]
	a.tailLabel = tailLabel

	argMap := make(map[string]*arg, 32)
	for i, _ := range a.argSlice {
		arg := &a.argSlice[i]
		argMap[fmt.Sprintf("-%c", arg.letter)] = arg
		argMap["--"+arg.name] = arg
	}

	for i, osarg := range osargs {
		if i == 0 {
			continue
		}
		arg, found := argMap[osarg]
		if found {
			if arg.resolved {
				a.FailWith(fmt.Sprintf("Duplicate option defined: %s", osarg))
			}
			if arg.outputBool != nil {
				*arg.outputBool = true
			} else {
				// end of slice or next element is an option
				if i == len(osargs)-1 || strings.HasPrefix(osargs[i+1], "-") {
					a.FailWith(fmt.Sprintf("Value expected for: %s", osarg))
				}
				value := osargs[i+1]

				if arg.outputString != nil {
					*arg.outputString = value
				}
				if arg.outputInt != nil {
					i, err := strconv.Atoi(value)
					if err != nil {
						a.FailWith(fmt.Sprintf("Option must be a number: %s", osarg))
					}
					*arg.outputInt = i
				}
				osargs[i+1] = "" // remove value
			}
			osargs[i] = "" // remove option
			arg.resolved = true
		} else if strings.HasPrefix(osarg, "-") {
			a.FailWith(fmt.Sprintf("Unknown option: %s", osarg))
		}
	}

	// set defaults and check mandatory options
	for i, _ := range a.argSlice {
		arg := &a.argSlice[i]
		if !arg.resolved && arg.required {
			a.FailWith(fmt.Sprintf("Value expected for: %s", arg.name))
		}
		if !arg.resolved && arg.outputBool == nil {
			if arg.outputString != nil {
				*arg.outputString = arg.dephault
			}
			if arg.outputInt != nil {
				i, _ := strconv.Atoi(arg.dephault)
				*arg.outputInt = i
			}
		}
	}

	// as options are processed they are removed from the slice. whatever remains is the tail.
	tail := []string{}
	for i, osarg := range osargs {
		if i == 0 {
			continue
		}
		if len(osarg) != 0 {
			tail = append(tail, osarg)
		}
	}

	if requireTail && len(tail) == 0 {
		a.FailWith("")
	}
	return tail
}

func (a *args) StringArg(letter byte, name string, dephault string, required bool, description string, choices []string, output *string) {
	if dephault != "" && required {
		shenanigans("Naughty args config: required and default are mutually exclusive: %s\n", name)
	}
	arg := arg{
		letter:       letter,
		name:         name,
		required:     required,
		dephault:     dephault,
		description:  description,
		choices:      choices,
		outputString: output,
	}
	a.argSlice = append(a.argSlice, arg)
}

func (a *args) BoolArg(letter byte, name string, description string, output *bool) {
	arg := arg{
		letter:      letter,
		name:        name,
		description: description,
		outputBool:  output,
	}
	a.argSlice = append(a.argSlice, arg)
}

func (a *args) IntArg(letter byte, name string, description string, dephault int, output *int) {
	arg := arg{
		letter:      letter,
		name:        name,
		description: description,
		dephault:    strconv.Itoa(dephault),
		outputInt:   output,
	}
	a.argSlice = append(a.argSlice, arg)
}

func (a *args) FailWith(error string) {
	a.PrintUsage()
	if error != "" {
		fmt.Printf("\n%s\n\n", error)
	}
	os.Exit(1)
}

func (a *args) PrintUsage() {
	fmt.Printf("\nUsage:\n    %s [options] %s\n\nOptions:\n", a.program, a.tailLabel)

	// setup format based on the longest option name
	l := 8
	for _, a := range a.argSlice {
		if len(a.name) > l {
			l = len(a.name)
		}
	}
	format := fmt.Sprintf("    -%%c  --%%-%ds %%-%ds  %%s", l, l+2)

	for _, a := range a.argSlice {
		i := ""
		if a.outputString != nil {
			i = fmt.Sprintf("<%s>", a.name)
		}
		fmt.Printf(format, a.letter, a.name, i, a.description)
		if len(a.choices) != 0 {
			fmt.Print(" [ ")
			for i, c := range a.choices {
				if i != 0 {
					fmt.Print(", ")
				}
				fmt.Print(c)
			}
			fmt.Print(" ]")
		}
		if a.dephault != "" {
			fmt.Print(" default [ ")
			fmt.Print(a.dephault)
			fmt.Print(" ]")
		}
		if a.required {
			fmt.Print(" (required)")
		}
		fmt.Println()
	}
	fmt.Println()
}
