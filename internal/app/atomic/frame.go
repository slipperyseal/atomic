package atomic

// tracks register usage and stack frame information for functions and scopes within them
type frame struct {
	parent    *frame
	values    map[string]register
	allocated map[int]string
	ref       *reference
}

func newFrame(ref *reference) *frame {
	f := frame{
		values:    make(map[string]register),
		allocated: make(map[int]string),
		ref:       ref,
	}
	return &f
}

func (f *frame) push() *frame {
	nf := newFrame(f.ref)
	nf.parent = f
	return nf
}

func (f *frame) pushParameter(p *profile, name string) register {
	for _, r := range p.registers {
		if r.param {
			_, allocated := f.allocated[r.index]
			if !allocated {
				f.values[name] = r
				f.allocated[r.index] = name
				return r
			}
		}
	}
	shenanigans("Unable to add param register for %s", name)
	return register{}
}

// finds or allocates a register for a value, returning true if the value already existed
func (f *frame) registerForValue(p *profile, name string) (register, bool) {
	r, exists := f.values[name]
	if exists {
		return r, true
	}

	for _, r := range p.registers {
		_, inUse := f.allocated[r.index]
		if !inUse {
			f.values[name] = r
			f.allocated[r.index] = name
			return r, false
		}
	}
	shenanigans("Unable to resolve register for %s", name)
	return register{}, false
}

func (f *frame) releaseValue(name string) bool {
	r, found := f.values[name]
	if found {
		delete(f.values, name)
		delete(f.allocated, r.index)
		return true
	}

	if f.parent != nil {
		return f.parent.releaseValue(name)
	}
	return false
}
