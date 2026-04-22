package pagination

func New(current, total int) Meta {
	var next *int
	last := current >= total

	if !last {
		n := current + 1
		next = &n
	}

	return Meta{
		Current: current,
		Next:    next,
		Total:   total,
		Last:    last,
	}
}
