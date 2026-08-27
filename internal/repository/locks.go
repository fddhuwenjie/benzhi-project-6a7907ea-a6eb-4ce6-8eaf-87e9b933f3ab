package repository

import "sync"

type keyedLocks struct {
	mu    sync.Mutex
	locks map[string]*lockRef
}

type lockRef struct {
	mu   sync.Mutex
	refs int
}

func newKeyedLocks() *keyedLocks { return &keyedLocks{locks: make(map[string]*lockRef)} }

func (k *keyedLocks) lock(key string) func() {
	k.mu.Lock()
	ref := k.locks[key]
	if ref == nil {
		ref = &lockRef{}
		k.locks[key] = ref
	}
	ref.refs++
	k.mu.Unlock()
	ref.mu.Lock()
	return func() {
		ref.mu.Unlock()
		k.mu.Lock()
		ref.refs--
		if ref.refs == 0 {
			delete(k.locks, key)
		}
		k.mu.Unlock()
	}
}
