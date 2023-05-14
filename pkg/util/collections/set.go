package collections

import (
	"fmt"
	"sort"
	"strings"
)

type Set[T comparable] map[T]struct{}

func NewSet[T comparable](values ...T) Set[T] {
	s := make(Set[T], len(values))
	return s
}

func (s Set[T]) Add(values ...T) {
	for _, val := range values {
		s[val] = struct{}{}
	}
}

func (s Set[T]) AsSlice() []T {
	slice := make([]T, 0, len(s))
	for val := range s {
		slice = append(slice, val)
	}
	return slice
}

func (s *Set[T]) String() string {
	ss := make([]string, 0, len(*s))
	for val := range *s {
		ss = append(ss, fmt.Sprint(val))
	}
	sort.Strings(ss)
	return strings.Join(ss, ",")
}
