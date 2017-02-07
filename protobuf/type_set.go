package protobuf

// TypeSet represents a set of packages and their types.
type TypeSet map[string]map[string]struct{}

// NewTypeSet returns a new, empty TypeSet.
func NewTypeSet() TypeSet {
	return TypeSet{}
}

// Add adds an element with the given package path and name to the set. Returns
// whether the element was or not added.
func (ts TypeSet) Add(pkg, name string) bool {
	if _, ok := ts[pkg]; !ok {
		ts[pkg] = make(map[string]struct{}, 1)
	}

	if _, ok := ts[pkg][name]; ok {
		return false
	}

	ts[pkg][name] = struct{}{}
	return true
}

// Contains checks if the given pkg and name is in the type set.
func (ts TypeSet) Contains(pkg, name string) bool {
	if p, ok := ts[pkg]; ok {
		_, ok = p[name]
		return ok
	}
	return false
}

// Len returns the total number of elements in the set.
func (ts TypeSet) Len() (c int) {
	for _, v := range ts {
		c += len(v)
	}
	return
}
