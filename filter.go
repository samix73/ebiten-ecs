package ecs

// Filter represents a predicate function for filtering entities based on component values
type Filter[C any] func(*C) bool

// And combines multiple filters with logical AND
func And[C any](filters ...Filter[C]) Filter[C] {
	return func(component *C) bool {
		for _, filter := range filters {
			if !filter(component) {
				return false
			}
		}
		return true
	}
}

// Or combines multiple filters with logical OR
func Or[C any](filters ...Filter[C]) Filter[C] {
	return func(component *C) bool {
		for _, filter := range filters {
			if filter(component) {
				return true
			}
		}
		return false
	}
}

// Not negates a filter
func Not[C any](filter Filter[C]) Filter[C] {
	return func(component *C) bool {
		return !filter(component)
	}
}
