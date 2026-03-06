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
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
		require.NoError(t, l.Lock(t.Context(), "test"))
		lockState := l.locks["test"]

		require.EqualValues(t, 0, lockState.count())

		// Start a goroutine that will wait for the lock
		lockAcquired := false
		go func() {
			assert.NoError(t, l.Lock(t.Context(), "test"))
			lockAcquired = true
		}()

		// Wait for goroutine to enter waiting state
		synctest.Wait()
		require.EqualValues(t, 1, lockState.count(), "should have one waiter")
		require.False(t, lockAcquired, "lock should not have been acquired while still held")

		// Release the lock - waiting goroutine should now acquire it
		l.Unlock("test")
		synctest.Wait()

		require.True(t, lockAcquired)
		require.EqualValues(t, 0, lockState.count())
	})
}

func TestLockerUnlock(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := New()

		require.NoError(t, l.Lock(t.Context(), "test"))
		l.Unlock("test")

		require.PanicsWithValue(t, "no such lock: test", func() {
			l.Unlock("test")
		})

		require.NoError(t, l.Lock(t.Context(), "test"))
	})
}

func TestLockerConcurrency(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := New()

		const numWorkers = 10_000
		results := make([]bool, numWorkers)

		for i := range numWorkers {
			go func() {
				assert.NoError(t, l.Lock(t.Context(), "test"))
				results[i] = true
				l.Unlock("test")
			}()
		}

		synctest.Wait()

		for i := range numWorkers {
			require.True(t, results[i], "worker %d should have acquired lock", i)
		}

		// Since everything has unlocked the map should be empty.
		require.Empty(t, l.locks)
	})
}

func TestLockerContextCanceled(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		l := New()

		err := l.Lock(ctx, "test")
		require.ErrorIs(t, context.Canceled, err)
	})
}

func TestLockerContextDeadlineExceeded(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 1*time.Millisecond)
		defer cancel()

		l := New()
		require.NoError(t, l.Lock(ctx, "test"))

		err := l.Lock(ctx, "test")
		require.ErrorIs(t, context.DeadlineExceeded, err)

		require.NotPanics(t, func() { l.Unlock("test") })
	})
}

func TestLockerMultipleKeysIsolation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := New()

		require.NoError(t, l.Lock(t.Context(), "key1"))
		require.EqualValues(t, 0, l.locks["key1"].count())

		// Should be able to lock key2 immediately (different key)
		require.NoError(t, l.Lock(t.Context(), "key2"))
		require.EqualValues(t, 0, l.locks["key2"].count())

		l.Unlock("key1")
		l.Unlock("key2")
		synctest.Wait()

		require.Empty(t, l.locks)
	})
}

func TestLockerContextCanceledWhileWaiting(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := New()
		require.NoError(t, l.Lock(t.Context(), "test"))

		ctx, cancel := context.WithCancel(t.Context())

		lockCanceled := false
		go func() {
			err := l.Lock(ctx, "test")
			assert.ErrorIs(t, err, context.Canceled)
			lockCanceled = true
		}()

		// Wait for goroutine to enter waiting state
		synctest.Wait()
		require.EqualValues(t, 1, l.locks["test"].count(), "should have one waiter")

		// Cancel while waiting
		cancel()
		synctest.Wait()

		require.True(t, lockCanceled)

		// Lock should still exist (held by main goroutine)
		require.EqualValues(t, 0, l.locks["test"].count())

		// Unlock and verify cleanup
		l.Unlock("test")
		synctest.Wait()
		require.Empty(t, l.locks)
	})
}
