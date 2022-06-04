package atomic

import "C"
import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"strconv"
	"strings"
	"unicode"
)

type parser struct {
	reader   *bufio.Reader
	filename string
	ast      ast
	astHash  string
	baseNode node
	lineNum  int
	verbose  bool
}

// minimal viable recursive parser - please don't take it very seriously
func (p *parser) parseSource(a *ast) {
	for {
		line, _, err := p.reader.ReadLine()
		if err == io.EOF {
			break
		} else if err != nil {
			shenanigans("Error reading source %v", err)
		}
		p.lineNum++
		// remove comments
		rawLine := strings.Split(string(line), ";")[0]
		tokens := strings.Fields(rawLine)
		if len(tokens) > 0 {
			token := tokens[0]
			switch p.baseNode.(type) {
			default: // no base node
				switch token {
				case "package:":
					a.append(&packageNode{
						name: tokens[1],
					})
				case "type:":
					as := a.append(&structNode{
						name: tokens[1],
					})
					p.baseNode = as.node
					p.parseSource(as)
					p.baseNode = nil
				case "function:":
					as := a.append(&functionNode{
						name: tokens[1],
					})
					p.baseNode = as.node
					p.parseSource(as)
					p.baseNode = nil
				default:
					p.syntaxError("Unknown base token: %s", strings.TrimSpace(rawLine))
				}
			case *functionNode:
				switch token {
				case "{":
					as := a.append(&scopeNode{})
					p.parseSource(as)
				case "}":
					a.resolve()
					return
				case ">":
					a.append(&inputNode{
						name: tokens[1],
					})
				case "+":
					a.append(&populateNode{
						name: tokens[1],
					})
				case "loop:":
					count, err := strconv.Atoi(tokens[1])
					if err != nil {
						p.syntaxError("Counter value not a number")
					}
					as := a.append(&loopNode{
						count: count,
					})
					p.parseSource(as)
				default:
					p.syntaxError("Unknown function token: %s", strings.TrimSpace(rawLine))
				}
			case *structNode:
				switch token {
				case "}":
					a.resolve()
					return
				}
				if unicode.IsLetter(rune(token[0])) {
					a.append(&fieldNode{
						name: token,
						typ:  tokens[1],
					})
				} else {
					p.syntaxError("Struct name must start with a letter: %s", token)
				}
			}
		}
	}
	a.resolve()
}

func (p *parser) syntaxError(format string, a ...interface{}) {
	shenanigans("%s line %d: %s", p.filename, p.lineNum, fmt.Sprintf(format, a...))
}

func (as *ast) printNode(w io.Writer, indent int) {
	for _, a := range as.sub {
		for i := 0; i <= indent; i++ {
			fmt.Fprintf(w, "  ")
		}
		if a.node != nil {
			fmt.Fprintf(w, "%s %+v\n", reflect.TypeOf(a.node), a.node)
		}
		a.printNode(w, indent+1)
	}
}

func (p *parser) calculateAstHash() {
	out := ioutil.Discard
	if p.verbose {
		out = os.Stdout
	}
	hashOut := newHashWriter(out)
	bufOut := bufio.NewWriter(hashOut)
	p.ast.printNode(bufOut, 0)
	bufOut.Flush()
	p.astHash = hashOut.result()
}

func parseFile(o options, filename string, channel chan ast) {
	file, err := os.Open(filename)
	if err != nil {
		shenanigans("Failed to open file: %s %v", filename, err)
	}
	defer file.Close()

	p := parser{
		reader:   bufio.NewReader(file),
		filename: filename,
		verbose:  o.verbose,
	}
	p.parseSource(&p.ast)
	p.calculateAstHash()
	if p.verbose {
		fmt.Printf("ast hash: %s\n", p.astHash)
	}
	channel <- p.ast
}
