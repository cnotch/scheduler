// Copyright (c) 2019,CAO HONGJU. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package scheduler

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const oneSecond = 1*time.Second + 10*time.Millisecond

func TestScheduler_Shutdown(t *testing.T) {
	t.Run("Scheduler.Shutdown", func(t *testing.T) {
		s := New()
		assert.False(t, s.terminated)
		s.ShutdownAndWait()
		assert.True(t, s.terminated)
	})
}

func TestScheduler_ScheduleAfterShutdown(t *testing.T) {
	t.Run("Scheduler.ScheduleAfterShutdown", func(t *testing.T) {
		s := New()
		var counter int32

		s.PeriodFunc(0, time.Second, func() {
			atomic.AddInt32(&counter, 1)
		})
		<-time.After(2 * oneSecond)
		s.ShutdownAndWait()
		want := int32(3)
		got := atomic.LoadInt32(&counter)
		assert.EqualValues(t, want, got)

		_, err := s.PeriodFunc(0, time.Second, func() {
			atomic.AddInt32(&counter, 1)
		})
		assert.NotNil(t, err)
	})
}

func TestScheduler_After(t *testing.T) {
	t.Run("Scheduler.After", func(t *testing.T) {
		s := New()
		defer s.Shutdown()
		out := make(chan bool, 1)
		mj, _ := s.AfterFunc(time.Millisecond*10, func() {
			out <- true
		})
		v := <-out
		assert.True(t, v)
		mj.Cancel()
	})
}

func TestScheduler_Period(t *testing.T) {
	t.Run("Scheduler.Period", func(t *testing.T) {
		s := New()
		defer s.Shutdown()
		var counter int32

		mj, _ := s.PeriodFunc(0, time.Second, func() {
			atomic.AddInt32(&counter, 1)
		})

		<-time.After(oneSecond)
		want := int32(2)
		got := atomic.LoadInt32(&counter)
		assert.EqualValues(t, want, got)

		mj.Cancel()

		<-time.After(oneSecond * 2)
		got = atomic.LoadInt32(&counter)
		assert.EqualValues(t, want, got)
	})
}

func TestScheduler_Cron(t *testing.T) {
	t.Run("Scheduler.Cron", func(t *testing.T) {
		s := New()
		defer s.Shutdown()

		wg := &sync.WaitGroup{}
		wg.Add(1)
		mj, _ := s.CronFunc("* * * * * ?", func() {
			wg.Done()
		})

		select {
		case <-time.After(oneSecond):
			t.Fatal("expected job runs")
		case <-wait(wg):
		}
		mj.Cancel()
	})
}

func TestScheduler_PeriodAndPanic(t *testing.T) {
	t.Run("Scheduler.PeriodAndPanic", func(t *testing.T) {
		s := New()
		defer s.Shutdown()
		assert.NotPanics(t, func() {
			var counter int32

			mj, _ := s.PeriodFunc(0, time.Millisecond*10, func() {
				atomic.AddInt32(&counter, 1)
				panic("test")
			})

			for atomic.LoadInt32(&counter) <= 10 {
			}

			mj.Cancel()

			<-time.After(100*time.Millisecond)
			want := atomic.LoadInt32(&counter)
			<-time.After(100*time.Millisecond)
			got := atomic.LoadInt32(&counter)
			assert.EqualValues(t, want, got)
		})
	})
}

func wait(wg *sync.WaitGroup) chan bool {
	ch := make(chan bool)
	go func() {
		wg.Wait()
		ch <- true
	}()
	return ch
}
