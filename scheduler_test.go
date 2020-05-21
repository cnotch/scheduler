// Copyright (c) 2019,CAO HONGJU. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package scheduler

import (
	"sort"
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

		mj, _ := s.PeriodFunc(0, time.Second, func() {
			atomic.AddInt32(&counter, 1)
		}, nil)
		<-time.After(2 * oneSecond)
		s.ShutdownAndWait()
		want := int32(3)
		got := atomic.LoadInt32(&counter)
		assert.EqualValues(t, want, got)

		mj.Cancel() // after shudown

		_, err := s.PeriodFunc(0, time.Second, func() {
			atomic.AddInt32(&counter, 1)
		}, nil)
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
		}, nil)
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
		}, nil)

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
		}, nil)

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
			}, nil)

			for atomic.LoadInt32(&counter) <= 10 {
			}

			mj.Cancel()

			<-time.After(100 * time.Millisecond)
			want := atomic.LoadInt32(&counter)
			<-time.After(100 * time.Millisecond)
			got := atomic.LoadInt32(&counter)
			assert.EqualValues(t, want, got)
		})
	})
}

func TestScheduler_PeriodAndPanicCancel(t *testing.T) {
	t.Run("Scheduler.PeriodAndPanic", func(t *testing.T) {
		var panicRecv interface{}
		s := New(WithPanicHandler(func(job *ManagedJob, r interface{}) {
			panicRecv = r
			job.Cancel()
		}))

		defer s.Shutdown()
		assert.NotPanics(t, func() {
			var counter int32

			mj, _ := s.PeriodFunc(0, time.Millisecond*10, func() {
				atomic.AddInt32(&counter, 1)
				panic("test")
			}, nil)
			<-time.After(100 * time.Millisecond)
			got := atomic.LoadInt32(&counter)
			assert.EqualValues(t, 1, got)
			assert.Equal(t, "test", panicRecv)
			assert.Equal(t, 0, s.Count())
			assert.Equal(t, -1, mj.index)

			mj.Cancel() // multiple calls
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

func TestScheduler_Jobs(t *testing.T) {
	wg := &sync.WaitGroup{}
	wg.Add(1)

	schd := New()
	defer schd.Shutdown()

	schd.PeriodFunc(2*time.Second, 2*time.Second, func() { wg.Done() }, nil)

	// Cron should fire in 2 seconds. After 1 second, call Entries.
	select {
	case <-time.After(oneSecond):
		schd.Jobs()
	}

	// Even though Entries was called, the cron should fire at the 2 second mark.
	select {
	case <-time.After(oneSecond):
		t.Error("expected job runs at 2 second mark")
	case <-wait(wg):
	}
}

func TestScheduler_MultipleJobs(t *testing.T) {
	pickTags := func(jobs []*ManagedJob) []string {
		tags := make([]string, len(jobs))
		for i, job := range jobs {
			tags[i] = job.Tag().(string)
		}
		sort.Strings(tags)
		return tags
	}

	wg := &sync.WaitGroup{}
	wg.Add(2)

	schd := New()

	var counter int32

	schd.CronFunc("0 0 0 1 1 ?", func() {}, "job1")
	schd.CronFunc("* * * * * ?", func() { wg.Done(); atomic.AddInt32(&counter, 1) }, "job2")
	mjob1, _ := schd.CronFunc("* * * * * ?", func() { t.Fatal() }, "job3")
	mjob2, _ := schd.CronFunc("* * * * * ?", func() { t.Fatal() }, "job4")
	schd.CronFunc("0 0 0 31 12 ?", func() {}, "job5")
	schd.CronFunc("* * * * * ?", func() { wg.Done(); atomic.AddInt32(&counter, 1) }, "job6")

	tags := pickTags(schd.Jobs())
	assert.Equal(t, []string{"job1", "job2", "job3", "job4", "job5", "job6"}, tags)

	mjob1.Cancel()
	tags = pickTags(schd.Jobs())
	assert.Equal(t, []string{"job1", "job2", "job4", "job5", "job6"}, tags)

	mjob2.Cancel()
	tags = pickTags(schd.Jobs())
	assert.Equal(t, []string{"job1", "job2", "job5", "job6"}, tags)

	select {
	case <-time.After(oneSecond):
		t.Error("expected job run in proper order")
	case <-wait(wg):
	}

	schd.ShutdownAndWait()
	atomic.StoreInt32(&counter, 0)
	jobs := schd.Jobs()
	assert.Equal(t, 0, len(jobs))
	<-time.After(oneSecond * 2)
	assert.Equal(t, int32(0), atomic.LoadInt32(&counter))
}

func TestManagedJob(t *testing.T) {
	schd := New()
	defer schd.Shutdown()
	mjob, _ := schd.CronFunc("* * * * * ?", func() {}, "tag")
	prev := time.Time{}.In(time.Local)
	next := time.Now()
	next = time.Date(next.Year(), next.Month(), next.Day(), next.Hour(), next.Minute(), next.Second(), 0, next.Location())
	for i := 0; i < 5; i++ {
		next = next.Add(time.Second)
		p := mjob.PrevTime()
		n := mjob.NextTime()
		assert.Equal(t, prev, p)
		assert.Equal(t, next, n)
		<-time.After(oneSecond)
		prev = next
	}
}
