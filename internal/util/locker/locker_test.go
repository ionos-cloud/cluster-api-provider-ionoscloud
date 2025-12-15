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

package locker

import (
	"context"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper to create a locked locker for testing.
func newLockedLocker(t *testing.T, key string) *Locker {
	t.Helper()
	l := New()
	require.NoError(t, l.Lock(context.Background(), key))
	return l
}

// Helper to verify lock exists with expected waiters.
func requireLockState(t *testing.T, l *Locker, key string, expectedWaiters int32) {
	t.Helper()
	require.Contains(t, l.locks, key, "lock %q should exist", key)
	require.EqualValues(t, expectedWaiters, l.locks[key].count(), "lock %q should have %d waiters", key, expectedWaiters)
}

func TestNew(t *testing.T) {
	locker := New()
	require.NotNil(t, locker)
	require.NotNil(t, locker.locks)
}

func TestLockWithCounter(t *testing.T) {
	lwc := &lockWithCounter{}
	require.EqualValues(t, 0, lwc.count())

	lwc.inc()
	require.EqualValues(t, 1, lwc.count())

	lwc.dec()
	require.EqualValues(t, 0, lwc.count())
}

func TestLockerLock(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := New()
		require.NoError(t, l.Lock(context.Background(), "test"))
		lockState := l.locks["test"]

		// Verify initial state: lock is held, no waiters
		require.EqualValues(t, 0, lockState.count())

		// Start a goroutine that will wait for the lock
		lockAcquired := make(chan struct{})
		go func(t *testing.T) {
			assert.NoError(t, l.Lock(context.Background(), "test"))
			close(lockAcquired)
		}(t)

		// Wait for goroutine to enter waiting state (deterministic with synctest)
		synctest.Wait()
		require.EqualValues(t, 1, lockState.count(), "should have one waiter")

		// Verify the waiting goroutine hasn't acquired the lock yet
		select {
		case <-lockAcquired:
			t.Fatal("lock should not have been acquired while still held")
		default:
		}

		// Release the lock - waiting goroutine should now acquire it
		l.Unlock("test")
		<-lockAcquired
		synctest.Wait()

		// Verify final state: no waiters
		require.EqualValues(t, 0, lockState.count())
	})
}

func TestLockerUnlock(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := New()

		require.NoError(t, l.Lock(context.Background(), "test"))
		l.Unlock("test")

		require.PanicsWithValue(t, "no such lock: test", func() {
			l.Unlock("test")
		})

		go func() {
			assert.NoError(t, l.Lock(context.Background(), "test"))
		}()
		synctest.Wait()
	})
}

func TestLockerConcurrency(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := New()

		var wg sync.WaitGroup
		for range 10_000 {
			wg.Go(func() {
				require.NoError(t, l.Lock(context.Background(), "test"))
				// If there is a concurrency issue, it will very likely become visible here.
				l.Unlock("test")
			})
		}

		wg.Wait()
		synctest.Wait()

		// Since everything has unlocked the map should be empty.
		require.Empty(t, l.locks)
	})
}

func TestLockerContextCanceled(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		l := New()

		err := l.Lock(ctx, "test")
		require.ErrorIs(t, context.Canceled, err)
		synctest.Wait()
	})
}

func TestLockerContextDeadlineExceeded(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		l := New()
		require.NoError(t, l.Lock(ctx, "test"))

		lockFailed := make(chan struct{})
		go func() {
			err := l.Lock(ctx, "test")
			assert.ErrorIs(t, context.DeadlineExceeded, err)
			close(lockFailed)
		}()

		<-lockFailed
		synctest.Wait()

		require.NotPanics(t, func() { l.Unlock("test") })
	})
}

// TestLockerMultipleKeysIsolation verifies that locks for different keys don't interfere with each other.
func TestLockerMultipleKeysIsolation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := New()

		// Lock key1
		require.NoError(t, l.Lock(context.Background(), "key1"))
		requireLockState(t, l, "key1", 0)

		// Should be able to lock key2 immediately (different key)
		key2Acquired := make(chan struct{})
		go func() {
			assert.NoError(t, l.Lock(context.Background(), "key2"))
			close(key2Acquired)
		}()

		<-key2Acquired
		synctest.Wait()

		// Both keys should be locked
		requireLockState(t, l, "key1", 0)
		requireLockState(t, l, "key2", 0)

		// Unlock both
		l.Unlock("key1")
		l.Unlock("key2")
		synctest.Wait()

		// Both locks should be cleaned up
		require.Empty(t, l.locks)
	})
}

// TestLockerMultipleWaiters verifies that multiple goroutines can wait for the same lock.
func TestLockerMultipleWaiters(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := newLockedLocker(t, "test")
		const numWaiters = 10

		var wg sync.WaitGroup
		results := make([]bool, numWaiters)

		// Start multiple waiters
		for i := range numWaiters {
			//nolint:copyloopvar // We want to use the loop variable in the goroutine
			i := i
			wg.Go(func() {
				//nolint:testifylint // Using assert in goroutine per go-require rule
				assert.NoError(t, l.Lock(context.Background(), "test"))
				results[i] = true
				l.Unlock("test")
			})
		}

		// Wait for all goroutines to enter waiting state
		synctest.Wait()
		require.EqualValues(t, numWaiters, l.locks["test"].count(), "should have %d waiters", numWaiters)

		// Release lock - all waiters should acquire it sequentially
		l.Unlock("test")
		wg.Wait()
		synctest.Wait()

		// All should have succeeded
		for i := range numWaiters {
			require.True(t, results[i], "waiter %d should have acquired lock", i)
		}

		// Lock should be cleaned up
		require.Empty(t, l.locks)
	})
}

// TestLockerContextCanceledWhileWaiting verifies context cancellation that happens while waiting (not before).
func TestLockerContextCanceledWhileWaiting(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := newLockedLocker(t, "test")

		ctx, cancel := context.WithCancel(context.Background())

		lockCanceled := make(chan struct{})
		go func() {
			err := l.Lock(ctx, "test")
			assert.ErrorIs(t, err, context.Canceled)
			close(lockCanceled)
		}()

		// Wait for goroutine to enter waiting state
		synctest.Wait()
		require.EqualValues(t, 1, l.locks["test"].count(), "should have one waiter")

		// Cancel while waiting
		cancel()
		<-lockCanceled
		synctest.Wait()

		// Lock should still exist (held by main goroutine)
		requireLockState(t, l, "test", 0)

		// Unlock and verify cleanup
		l.Unlock("test")
		synctest.Wait()
		require.Empty(t, l.locks)
	})
}

// TestLockerReacquisition verifies that a lock can be re-acquired after unlock.
func TestLockerReacquisition(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := New()

		// Lock, unlock, lock again
		require.NoError(t, l.Lock(context.Background(), "test"))
		l.Unlock("test")

		require.NoError(t, l.Lock(context.Background(), "test"))
		l.Unlock("test")

		// Lock should be cleaned up
		require.Empty(t, l.locks)
	})
}

// TestLockerCleanupAfterAllWaitersCancel verifies that lock is cleaned up when all waiters cancel.
func TestLockerCleanupAfterAllWaitersCancel(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := newLockedLocker(t, "test")

		ctx, cancel := context.WithCancel(context.Background())

		const numWaiters = 5
		var wg sync.WaitGroup
		for range numWaiters {
			wg.Go(func() {
				err := l.Lock(ctx, "test")
				require.ErrorIs(t, err, context.Canceled)
			})
		}

		// Wait for all goroutines to enter waiting state
		synctest.Wait()
		require.EqualValues(t, numWaiters, l.locks["test"].count(), "should have %d waiters", numWaiters)

		// Cancel all waiters
		cancel()
		wg.Wait()
		synctest.Wait()

		// Lock should still exist (held by main goroutine)
		requireLockState(t, l, "test", 0)

		// Unlock - now lock should be cleaned up
		l.Unlock("test")
		synctest.Wait()
		require.Empty(t, l.locks)
	})
}
