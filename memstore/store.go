package memstore

import (
	"cmp"
	"errors"
	"slices"
)

type Ider interface {
	GetID() string
}

type Store[T Ider] struct {
	items map[string]T
}

func New[T Ider]() *Store[T] {
	return &Store[T]{
		items: map[string]T{},
	}
}

var ErrDuplicate = errors.New("duplicate ID")

func (s *Store[T]) Create(item T) error {
	id := item.GetID()
	if _, ok := s.items[id]; ok {
		return ErrDuplicate
	}
	s.items[id] = item
	return nil
}

func (s *Store[T]) All() ([]T, error) {
	var items []T
	for _, item := range s.items {
		items = append(items, item)
	}

	slices.SortFunc(items, func(a, b T) int {
		return cmp.Compare(a.GetID(), b.GetID())
	})
	return items, nil
}

func (s *Store[T]) Delete(id string) error {
	delete(s.items, id)
	return nil
}
