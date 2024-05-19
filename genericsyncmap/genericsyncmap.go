package genericsync

import (
	"sync"
)

// Source: https://github.com/simplylib/genericsync/
// Map is a generic wrapper around sync.Map.
// Refer to sync.Map for documentation on how functions work.
type Map[K any, V any] struct {
	m sync.Map
}

func (m *Map[K, V]) Load(k K) (value V, ok bool) {
	var v any
	v, ok = m.m.Load(k)
	if !ok {
		return
	}
	return v.(V), ok
}

func (m *Map[K, V]) LoadOrStore(k K, v V) (actual V, loaded bool) {
	var a any
	a, loaded = m.m.LoadOrStore(k, v)
	if !loaded {
		return v, loaded
	}
	return a.(V), loaded
}

func (m *Map[K, V]) Store(k K, v V) {
	m.m.Store(k, v)
}

func (m *Map[K, V]) Delete(k K) {
	m.m.Delete(k)
}

func (m *Map[K, V]) LoadAndDelete(k K) (value V, loaded bool) {
	var v any
	v, loaded = m.m.LoadAndDelete(k)
	if !loaded {
		return
	}
	return v.(V), loaded
}

func (m *Map[K, V]) Range(f func(k K, v V) bool) {
	m.m.Range(func(key, value any) bool {
		if key != nil && value != nil {
			return f(key.(K), value.(V))
		}

		var k K
		if key != nil {
			k = key.(K)
		}

		var v V
		if value != nil {
			v = value.(V)
		}

		return f(k, v)
	})
}
