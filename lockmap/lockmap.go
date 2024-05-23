// Package lockmap implements a wrapper for map protected by a mutex.
package lockmap

import (
	"sync"
)

// LockMap is a generic wrapper around map with a lock attached.
// Dont forget to MakeLockMap[K, V]() or get panic'ed
type LockMap[K comparable, V any] struct {
	mu sync.Mutex
	m  map[K]V
}

// MakeLockMap make()s the underlying map ready for use.
// Just like a normal map, you have to make() it.
// It takes an optional parameter to an already existing
// map to use as initial map. You may not access it anymore
// after you have created the lockmap and only access it
// through LockMap's exposed functions.
func MakeLockMap[K comparable, V any](initialmap *map[K]V) *LockMap[K, V] {
	lm := LockMap[K, V]{}
	if initialmap != nil {
		lm.m = *initialmap
	} else {
		lm.m = make(map[K]V)
	}
	return &lm
}

// LockedLoad performs the following steps:
// - Locks the Map
// - Loads the Value
// - Unlocks the Map
func (lm *LockMap[K, V]) LockedLoad(k K) (V, bool) {
	lm.mu.Lock()
	value, ok := lm.m[k]
	lm.mu.Unlock()
	return value, ok
}

// LockedLoadAndDelete performs the following steps:
// - Locks the Map
// - Gets the Value
// - Deletes the Value
// - Unlocks the Map
func (lm *LockMap[K, V]) LockedLoadAndDelete(k K) (V, bool) {
	lm.mu.Lock()
	value, ok := lm.m[k]
	delete(lm.m, k)
	lm.mu.Unlock()
	return value, ok
}

// LockedLoadAndDelete performs the following steps:
// - Locks the Map
// - Deletes the Value
// - Unlocks the Map
func (lm *LockMap[K, V]) LockedDelete(k K) {
	lm.mu.Lock()
	delete(lm.m, k)
	lm.mu.Unlock()
}

// LockAndLoad performs the following steps:
// - Locks the Map
// - Loads the Value
func (lm *LockMap[K, V]) LockAndLoad(k K) (V, bool) {
	lm.mu.Lock()
	value, ok := lm.m[k]
	return value, ok
}

// StoreAndUnlock performs the following steps:
// - Stores the Value
// - Unlocks the Map
func (lm *LockMap[K, V]) StoreAndUnlock(k K, v V) {
	lm.m[k] = v
	lm.mu.Unlock()
}

// DeleteAndUnlock performs the following steps:
// - Deletes the Value
// - Unlocks the Map
func (lm *LockMap[K, V]) DeleteAndUnlock(k K) {
	delete(lm.m, k)
	lm.mu.Unlock()
}

// LockedRange performs the following steps:
// - Locks the Map
// - Loops until f returns false or all done
// - Unlocks the Map
func (lm *LockMap[K, V]) LockedRange(f func(k K, v V) bool) {
	lm.mu.Lock()
	for k, v := range lm.m {
		if f(k, v) == false {
			break
		}
	}
	lm.mu.Unlock()
}

// LockLoadWithUnlockerFunc performs the following steps:
//   - Locks the Map
//   - Loads the Value
//   - Returns an unlocker func which you should defer or
//     call when you are done with it.
func (lm *LockMap[K, V]) LockLoadWithUnlockerFunc(k K) (V, bool, func()) {
	lm.mu.Lock()
	value, ok := lm.m[k]
	return value, ok, func() {
		lm.mu.Unlock()
	}
}

// StoreWhenWithUnlocker is used when storing a value when
// the map was previously locked with LockLoadWithUnlockerFunc
func (lm *LockMap[K, V]) StoreWhenWithUnlocker(k K, v V) {
	lm.m[k] = v
}

// UnderlyingMap returns a pointer to the map that is protected.
// You are supposed to only read from the map. Unlock the map with
// the returned unlocker function after you are done reading.
func (lm *LockMap[K, V]) LockAndGetUnderlyingMapWithUnlocker() (*map[K]V, func()) {
	lm.mu.Lock()
	return &lm.m, func() {
		lm.mu.Unlock()
	}
}
