package server

import "sync"

type keyedLock struct {
	mu      sync.Mutex
	entries map[string]*keyedLockEntry
}

type keyedLockEntry struct {
	mu   sync.Mutex
	refs int
}

func newKeyedLock() *keyedLock {
	return &keyedLock{
		entries: map[string]*keyedLockEntry{},
	}
}

// Lock acquires an exclusive lock for the given key and returns a release func.
func (l *keyedLock) Lock(key string) func() {
	l.mu.Lock()
	e, ok := l.entries[key]
	if !ok {
		e = &keyedLockEntry{}
		l.entries[key] = e
	}
	e.refs++
	l.mu.Unlock()

	e.mu.Lock()

	return func() {
		e.mu.Unlock()

		l.mu.Lock()
		e.refs--
		if e.refs == 0 {
			delete(l.entries, key)
		}
		l.mu.Unlock()
	}
}
