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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withTimeout(t *testing.T, f func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		f()
		close(done)
	}()
	select {
	case <-time.After(1 * time.Second):
		t.Fatal("timed out")
	case <-done:
	}
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
	l := New()
	require.NoError(t, l.Lock(context.Background(), "test"))
	lwc := l.locks["test"]

	require.EqualValues(t, 0, lwc.count())

	chDone := make(chan struct{})
	go func(t *testing.T) {
		assert.NoError(t, l.Lock(context.Background(), "test"))
		close(chDone)
	}(t)

	chWaiting := make(chan struct{})
	go func() {
		for range time.Tick(1 * time.Millisecond) {
			if lwc.count() == 1 {
				close(chWaiting)
				break
			}
		}
	}()

	withTimeout(t, func() {
		<-chWaiting
	})

	select {
	case <-chDone:
		t.Fatal("lock should not have returned while it was still held")
	default:
	}

	l.Unlock("test")

	withTimeout(t, func() {
		<-chDone
	})

	require.EqualValues(t, 0, lwc.count())
}

func TestLockerUnlock(t *testing.T) {
	l := New()

	require.NoError(t, l.Lock(context.Background(), "test"))
	l.Unlock("test")

	require.PanicsWithValue(t, "no such lock: test", func() {
		l.Unlock("test")
	})

	withTimeout(t, func() {
		require.NoError(t, l.Lock(context.Background(), "test"))
	})
}

func TestLockerConcurrency(t *testing.T) {
	l := New()

	var wg sync.WaitGroup
	for i := 0; i <= 10_000; i++ {
		wg.Add(1)
		go func(t *testing.T) {
			assert.NoError(t, l.Lock(context.Background(), "test"))
			// If there is a concurrency issue, it will very likely become visible here.
			l.Unlock("test")
			wg.Done()
		}(t)
	}

	withTimeout(t, wg.Wait)

	// Since everything has unlocked the map should be empty.
	require.Empty(t, l.locks)
}

func TestLockerContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	l := New()

	withTimeout(t, func() {
		err := l.Lock(ctx, "test")
		require.ErrorIs(t, context.Canceled, err)
	})
}

func TestLockerContextDeadlineExceeded(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	l := New()
	require.NoError(t, l.Lock(ctx, "test"))

	withTimeout(t, func() {
		err := l.Lock(ctx, "test")
		require.ErrorIs(t, context.DeadlineExceeded, err)
	})

	require.NotPanics(t, func() { l.Unlock("test") })
}
