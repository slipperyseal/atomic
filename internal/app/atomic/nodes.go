package atomic

type packageNode struct {
	name string
}

func (n *packageNode) resolve(a *ast) func(f *frame, p *profile, as *asm) func() {
	return nil
}

type structNode struct {
	name   string
	fields []field
	size   int
}

type field struct {
	name string

	prim   primative
	offset int
}

func (n *structNode) resolve(a *ast) func(f *frame, p *profile, as *asm) func() {
	off := 0
	for _, s := range a.sub {
		switch s.node.(type) {
		case *fieldNode:
			fn := s.node.(*fieldNode)
			p, ok := primatives[fn.typ]
			if !ok {
				shenanigans("Unknown type " + fn.typ)
			}
			f := field{
				name:   fn.name,
				prim:   p,
				offset: off,
			}
			n.fields = append(n.fields, f)
			off += p.size
		}
	}
	return nil
}

type fieldNode struct {
	name string
	typ  string
}

func (n *fieldNode) resolve(a *ast) func(f *frame, p *profile, as *asm) func() {
	return nil
}

type functionNode struct {
	name string
}

func (n *functionNode) resolve(a *ast) func(f *frame, p *profile, as *asm) func() {
	return func(f *frame, p *profile, as *asm) func() {
		as.addSymbol("_"+n.name, true)
		return func() {
			as.addSymbol("exit_"+n.name, false)
			as.addOpCode(p.findReg("ret", p.findLinkRegister().index))
		}
	}
}

type scopeNode struct {
}

func (n *scopeNode) resolve(a *ast) func(f *frame, p *profile, as *asm) func() {
	a.deleteIfNoSubElements()
	a.collapseIfSingleSubElement()
	return nil
}

type loopNode struct {
	count int
}

func (n *loopNode) resolve(a *ast) func(f *frame, p *profile, as *asm) func() {
	// if single sub node is a loop, collapse loop and combine counts
	if len(a.sub) == 1 {
		s := &a.sub[0]
		switch s.node.(type) {
		case *loopNode:
			ln := s.node.(*loopNode)
			n.count *= ln.count
			a.sub = s.sub
		}
	}
	// if no sub ast or loop count is zero, nothing to do
	if len(a.sub) == 0 || n.count == 0 {
		a.node = nil
	}
	// if loop count is one, convert this to a simple scope
	if n.count == 1 {
		scope := &scopeNode{}
		a.node = scope
		scope.resolve(a)
	}
	return nil
}

type populateNode struct {
	name string
}

func (n *populateNode) resolve(a *ast) func(f *frame, p *profile, as *asm) func() {
	return func(f *frame, p *profile, as *asm) func() {
		targetStruct, ok := f.ref.structs[n.name]
		if !ok {
			shenanigans("Unable to resolve struct %s", n.name)
		}

		targetRegister, ok := f.registerForValue(p, n.name)
		if !ok {
			shenanigans("Unable to resolve %s", n.name)
		}
		tmp, _ := f.registerForValue(p, "tmp")

		// this method currently makes a lot of terrible assumptions but its ok for a demo
		for _, field := range targetStruct.fields {
			for _, sourceStruct := range f.ref.structs {
				if n.name != sourceStruct.name {
					for _, sourceField := range sourceStruct.fields {
						if sourceField.name == field.name {
							sourceRegister, ok := f.registerForValue(p, sourceStruct.name)
							if ok {
								as.addOpCode(p.find("ldr", "int").set("int", sourceField.offset/8, sourceRegister.index, tmp.index))
								as.addOpCode(p.find("str", "int").set("int", field.offset/8, targetRegister.index, tmp.index))
							}
						}
					}
				}
			}
		}
		f.releaseValue("tmp")
		return nil
	}
}

type inputNode struct {
	name string
}

func (n *inputNode) resolve(a *ast) func(f *frame, p *profile, as *asm) func() {
	return func(f *frame, p *profile, as *asm) func() {
		f.pushParameter(p, n.name)
		// move reg to self as non breaky way to see it doing something
		//as.addOpCode(p.find("mov", "dn").set("dn", r.index, r.index))
		return func() {
			f.releaseValue(n.name)
		}
	}
}
