package atomic

import (
	"fmt"
)

type ast struct {
	sub     []ast
	node    node
	emitter func(f *frame, p *profile, a *asm) func()
}

type node interface {
	resolve(a *ast) func(f *frame, p *profile, a *asm) func() // called on the final ascent of the ast, allowing nodes to collect data, optimise etc
}

type primative struct {
	name    string
	size    int
	integer bool
	boolean bool
	signed  bool
	float   bool
}

type reference struct {
	pack      string
	structs   map[string]*structNode
	functions map[string]*functionNode
}

func (a *ast) resolve() {
	compact := false
	for i, _ := range a.sub {
		s := &a.sub[i]
		s.emitter = s.node.resolve(s)
		if s.node == nil {
			compact = true
		}
	}
	if compact {
		n := 0
		for i, _ := range a.sub {
			s := &a.sub[i]
			if s.node != nil {
				na := &a.sub[n]
				na.sub = s.sub
				na.node = s.node
				n++
			}
		}
		a.sub = a.sub[:n]
	}
}

func (a *ast) emit(f *frame, p *profile, as *asm) func() {
	var closure func()
	if a.emitter != nil {
		closure = a.emitter(f, p, as)
	}

	var deferred []func()
	for i, _ := range a.sub {
		sa := &a.sub[i]
		dc := sa.emit(f, p, as)
		if dc != nil {
			deferred = append(deferred, dc)
		}
	}
	for _, dc := range deferred {
		dc()
	}

	if closure != nil {
		if len(a.sub) > 0 {
			// if node has sub nodes run the closure now, post sub nodes
			closure()
		} else {
			// otherwise return closure to the caller, deferred to the end of this node's peers
			return closure
		}
	}
	return nil
}

func (a *ast) deleteIfNoSubElements() {
	if len(a.sub) == 0 {
		a.node = nil
	}
}

func (a *ast) collapseIfSingleSubElement() {
	if len(a.sub) == 1 {
		s := &a.sub[0]
		a.sub = s.sub
		a.node = s.node
	}
}

func (a *ast) append(n node) *ast {
	as := ast{
		node: n,
	}
	a.sub = append(a.sub, as)
	return &a.sub[len(a.sub)-1]
}

func (p *primative) String() string {
	return fmt.Sprintf("%s:%d", p.name, p.size)
}

var primatives = map[string]primative{
	"[]": {
		name: "array",
		size: 4,
	},
	"string": {
		name: "string",
		size: 8, // assuming 64 bit architecture
	},
	"bool": {
		name:    "bool",
		size:    1,
		boolean: true,
	},
	"byte": {
		name:    "byte",
		size:    1,
		integer: true,
	},
	"short": {
		name:    "short",
		size:    2,
		integer: true,
	},
	"int": {
		name:    "int",
		size:    4,
		integer: true,
	},
	"long": {
		name:    "long",
		size:    8,
		integer: true,
	},
	"double": {
		name:  "double",
		size:  8,
		float: true,
	},
}

func (r reference) populate(a *ast) {
	for _, n := range a.sub {
		switch n.node.(type) {
		case *structNode:
			s := n.node.(*structNode)
			if _, dupe := r.structs[s.name]; dupe {
				shenanigans("Duplicate struct definition: %s", s.name)
			}
			r.structs[s.name] = s
		case *functionNode:
			f := n.node.(*functionNode)
			if _, dupe := r.structs[f.name]; dupe {
				shenanigans("Duplicate function definition: %s", f.name)
			}
			r.functions[f.name] = f
		case *packageNode:
			p := n.node.(*packageNode)
			if r.pack != "" {
				shenanigans("Package already defined: %s", r.pack)
			}
			r.pack = p.name
		}
	}
	if r.pack == "" {
		shenanigans("Package not defined")
	}
}
