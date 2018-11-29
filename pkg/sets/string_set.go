package sets

type StringSet struct {
	vals map[string]bool
}

func NewStringSet(strings []string) *StringSet {
	vals := make(map[string]bool)
	for _, s := range strings {
		vals[s] = true
	}
	return &StringSet{
		vals: vals,
	}
}

func (s *StringSet) Contains(val string) bool {
	return s.vals[val]
}

func (s *StringSet) ContainsAll(vals []string) bool {
	for _, val := range vals {
		if !s.Contains(val) {
			return false
		}
	}

	return true
}