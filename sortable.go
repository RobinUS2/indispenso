package main

type sortableSlice []*sortData

type sortData struct {
	val string
	obj interface{}
}

// Len is part of sort.Interface.
func (d sortableSlice) Len() int {
	return len(d)
}

// Swap is part of sort.Interface.
func (d sortableSlice) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}

// Less is part of sort.Interface. We use count as the value to sort by
func (d sortableSlice) Less(i, j int) bool {
	return d[i].val < d[j].val
}

// New
func newSortableSlice() sortableSlice {
	return make(sortableSlice, 0)
}
