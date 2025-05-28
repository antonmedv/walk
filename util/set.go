package util

type Set[T comparable] struct {
	// Using struct{} is more memory efficient than using interface{} or bool
	hashset map[T]struct{}
}

// NewSet Constructor
func NewSet[T comparable]() *Set[T] {
	return &Set[T]{
		hashset: make(map[T]struct{}),
	}
}

func (s *Set[T]) IsPresent(element T) bool {
	_, isPresent := s.hashset[element]
	return isPresent
}

func (s *Set[T]) Add(element T) {
	if s.IsPresent(element) {
		return
	}

	s.hashset[element] = struct{}{}
}

func (s *Set[T]) Remove(element T) {
	// Safe to delete even if the element is not present
	// https://go.dev/doc/effective_go#maps
	delete(s.hashset, element)
}

func (s *Set[T]) Clear() {
	clear(s.hashset)
}

func (s *Set[T]) Size() int {
	return len(s.hashset)
}

func (s *Set[T]) ToList() []T {
	list := make([]T, s.Size())
	i := 0
	for item, _ := range s.hashset {
		list[i] = item
		i++
	}

	return list
}
