/*
Copyright 2024 IONOS Cloud.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

/*
Package locker provides a mechanism for fine-grained locking.

In contrast to a sync.Mutex, the user must provide a key when locking and unlocking.

If a lock with a given key does not exist when Lock() is called, one is created.
Lock references are automatically cleaned up if nothing is waiting for the lock anymore.

Locking can be aborted on context cancellation.

This package is inspired by https://github.com/moby/locker, but uses a buffered channel instead of a sync.Mutex to allow
for context cancellation.
Even though the standard library does not offer to cancel locking a mutex, golang.org/x/sync/semaphore does for its
Acquire() method.
*/
package locker

import (
	"context"
	"sync"
	"sync/atomic"
)

// Locker offers thread-safe locking of entities represented by keys of type string.
type Locker struct {
	mu    sync.Mutex // protects the locks map
	locks map[string]*lockWithCounter
}

// New returns a new Locker.
func New() *Locker {
	locker := &Locker{
		locks: make(map[string]*lockWithCounter),
	}
	return locker
}

// lockWithCounter is used by Locker to represent a lock for a given key.
type lockWithCounter struct {
	// ch is a buffered channel of size 1 used as a mutex.
	// In contrast to a sync.Mutex, accessing the channel can be aborted upon context cancellation.
	ch chan struct{}
	// waiters is the number of callers waiting to acquire the lock.
	// This is Int32 instead of Uint32, so we can add -1 in dec().
	waiters atomic.Int32
}

// inc increments the number of waiters.
func (lwc *lockWithCounter) inc() {
	lwc.waiters.Add(1)
}

// dec decrements the number of waiters.
func (lwc *lockWithCounter) dec() {
	lwc.waiters.Add(-1)
}

// count gets the current number of waiters.
func (lwc *lockWithCounter) count() int32 {
	return lwc.waiters.Load()
}

// Lock locks the given key. Returns after acquiring lock or when given context is done.
// The only errors it can return stem from the context being done.
func (l *Locker) Lock(ctx context.Context, key string) error {
	l.mu.Lock()

	if err := ctx.Err(); err != nil {
		// ctx becoming done has "happened before" acquiring the lock, whether it became done before the call began or
		// while we were waiting for the mutex. We prefer to fail even if we could acquire the mutex without blocking.
		l.mu.Unlock()
		return err
	}

	lwc, exists := l.locks[key]
	if !exists {
		lwc = &lockWithCounter{
			ch: make(chan struct{}, 1),
		}
		l.locks[key] = lwc
	}
	// Increment the waiters while inside the mutex to make sure that the lock isn't deleted if Lock() and Unlock()
	// are called concurrently.
	lwc.inc()

	l.mu.Unlock()

	// Lock the key outside the mutex, so we don't block other operations.
	select {
	case lwc.ch <- struct{}{}:
		// Lock acquired, so we can decrement the number of waiters for this lock.
		lwc.dec()
		return nil
	case <-ctx.Done():
		// Locking aborted, so we can decrement the number of waiters for this lock.
		lwc.dec()
		// If there are no more waiters, we can delete the lock.
		l.mu.Lock()
		if lwc.count() == 0 {
			delete(l.locks, key)
		}
		l.mu.Unlock()

		return ctx.Err()
	}
}

// Unlock unlocks the given key. It panics if the key is not locked.
func (l *Locker) Unlock(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	lwc, exists := l.locks[key]
	if !exists {
		panic("no such lock: " + key)
	}
	<-lwc.ch

	// If there are no more waiters, we can delete the lock.
	if lwc.count() == 0 {
		delete(l.locks, key)
	}
}
