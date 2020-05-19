// Copyright (c) 2019,CAO HONGJU. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package scheduler

import (
	"container/heap"
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cnotch/scheduler/cron"
)

const (
	minInterval = time.Millisecond // minimum trigger interval
)

// PanicHandler is to handle panic caused by an asynchronous job.
type PanicHandler func(job *ManagedJob, r interface{})

// A Scheduler maintains a registry of Jobs.
// Once registered, the Scheduler is responsible for executing Jobs
// when their scheduled time arrives.
type Scheduler struct {
	wg           *sync.WaitGroup
	add          chan *ManagedJob
	remove       chan *ManagedJob
	panicHandler PanicHandler
	loc          *time.Location
	ctx          context.Context
	cancel       context.CancelFunc
	terminated   bool
	count        int64
}

// New returns a new Scheduler instance.
func New(options ...Option) *Scheduler {
	s := &Scheduler{
		wg:     &sync.WaitGroup{},
		add:    make(chan *ManagedJob),
		remove: make(chan *ManagedJob),
		loc:    time.Local,
	}

	for _, option := range options {
		option.apply(s)
	}

	if s.ctx == nil {
		s.ctx, s.cancel = context.WithCancel(context.Background())
	}

	if s.panicHandler == nil {
		s.panicHandler = func(job *ManagedJob, r interface{}) {
			fmt.Fprintf(os.Stderr, "[Tag]: %+v [Error]: %+v\n", job.tag, r)
		}
	}

	// start
	s.wg.Add(1)
	go s.run()
	return s
}

// AfterFunc posts the function f which will execute after specified delay.
// The function f executes only once, and then remove from the Scheduler.
func (s *Scheduler) AfterFunc(delay time.Duration, f func(), tag interface{}) (*ManagedJob, error) {
	return s.After(delay, JobFunc(f), tag)
}

// After posts the job which will execute after specified delay.
// The job executes only once, and then remove from the Scheduler.
func (s *Scheduler) After(delay time.Duration, job Job, tag interface{}) (*ManagedJob, error) {
	return s.Schedule(&afterSchedule{delay: delay}, job, tag)
}

// PeriodFunc posts the function f which will execute the first time
// at the specified delay, followed by a fixed period.
// If the execution time of f exceeds the period, there will
// be multiple instances of f running at the same time.
func (s *Scheduler) PeriodFunc(initialDelay, period time.Duration, f func(), tag interface{}) (*ManagedJob, error) {
	return s.Period(initialDelay, period, JobFunc(f), tag)
}

// Period posts the job which will execute the first time
// at the specified delay, followed by a fixed period.
// If the execution time of job exceeds the period, there will
// be multiple instances of job running at the same time.
func (s *Scheduler) Period(initialDelay, period time.Duration, job Job, tag interface{}) (*ManagedJob, error) {
	if period < time.Millisecond {
		return nil, errors.New("preiod must not be less than 1ms")
	}
	return s.Schedule(&periodSchedule{initialDelay: initialDelay, period: period}, job, tag)
}

// CronFunc posts the function f which will execute according to a cron expression.
func (s *Scheduler) CronFunc(cronExpr string, f func(), tag interface{}) (*ManagedJob, error) {
	return s.Cron(cronExpr, JobFunc(f), tag)
}

// Cron posts the job which will execute according to a cron expression.
func (s *Scheduler) Cron(cronExpr string, job Job, tag interface{}) (*ManagedJob, error) {
	cexp, err := cron.Parse(cronExpr)
	if err != nil {
		return nil, err
	}
	return s.Schedule(cexp, job, tag)
}

// ScheduleFunc posts the function f which will execute according to the specified schedule
func (s *Scheduler) ScheduleFunc(schedule Schedule, f func(), tag interface{}) (*ManagedJob, error) {
	return s.Schedule(schedule, JobFunc(f), tag)
}

// Schedule posts the job which will execute according to the specified schedule.
func (s *Scheduler) Schedule(schedule Schedule, job Job, tag interface{}) (mjob *ManagedJob, err error) {
	defer func() { // after terminated, add throw panic
		if r := recover(); r != nil {
			err = errors.New("scheduler is terminated")
		}
	}()

	j := &ManagedJob{
		tag:      tag,
		schelule: schedule,
		job:      job,
		remove:   s.remove,
	}

	j.next = j.schelule.Next(s.now())
	if j.next.IsZero() {
		return nil, errors.New("schedule is empty, never a scheduled time to arrive")
	}

	s.add <- j
	return j, nil
}

// Shutdown shutdowns scheduler.
func (s *Scheduler) Shutdown() {
	s.cancel()
}

// ShutdownAndWait  shutdowns scheduler and wait for all jobs to complete.
func (s *Scheduler) ShutdownAndWait() {
	s.cancel()
	s.wg.Wait()
}

// Terminated determines that the scheduler has terminated
func (s *Scheduler) Terminated() bool {
	return s.terminated
}

// Count returns jobs count.
func (s *Scheduler) Count() int {
	l := atomic.LoadInt64(&s.count)
	return int(l)
}

func (s *Scheduler) run() {
	defer s.wg.Done()

	jobs := make(jobQueue, 0, 16)
	for {
		atomic.StoreInt64(&s.count, int64(len(jobs)))

		d := time.Duration(100000 * time.Hour) // if there are no jobs
		if len(jobs) > 0 {
			d = jobs[0].next.Sub(s.now())
			if d < 0 {
				d = 0
			}
		}
		timer := time.NewTimer(d)

		select {
		case <-s.ctx.Done(): // exit Scheduler
			timer.Stop()
			s.internalClose()
			return

		case now := <-timer.C:
			now = now.In(s.loc)
			s.runExpiredJobs(now, &jobs)

		case newJ := <-s.add:
			timer.Stop()
			heap.Push(&jobs, newJ)

		case removeJ := <-s.remove: // 删除作业
			timer.Stop()
			s.removeJob(removeJ, &jobs)
		}
	}
}

func (s *Scheduler) runExpiredJobs(now time.Time, jobs *jobQueue) {
	for len(*jobs) > 0 {
		j := (*jobs)[0]
		if j.next.After(now) {
			break
		}

		s.wg.Add(1)
		go s.safeRun(j)

		next := j.schelule.Next(j.next)
		if next.IsZero() {
			heap.Pop(jobs)
		} else {
			jobs.updateNext(j, next)
		}
	}
}

func (s *Scheduler) safeRun(j *ManagedJob) {
	defer func() {
		s.wg.Done()
		if r := recover(); r != nil {
			s.panicHandler(j, r)
		}
	}()
	j.job.Run()
}

func (s *Scheduler) removeJob(removeJ *ManagedJob, jobs *jobQueue) {
	if removeJ.index < 0 || removeJ.index >= len(*jobs) {
		return
	}

	if removeJ == (*jobs)[removeJ.index] {
		heap.Remove(jobs, removeJ.index)
	}
}

func (s *Scheduler) internalClose() {
	s.terminated = true
	close(s.add)
	close(s.remove)
	atomic.StoreInt64(&s.count, 0)
}

func (s *Scheduler) now() time.Time {
	return time.Now().In(s.loc)
}

type afterSchedule struct {
	called bool
	delay  time.Duration
}

func (at *afterSchedule) Next(t time.Time) time.Time {
	if at.called {
		return time.Time{}
	}

	at.called = true
	return t.Add(at.delay)
}

type periodSchedule struct {
	called               bool
	initialDelay, period time.Duration
}

func (pt *periodSchedule) Next(t time.Time) time.Time {
	d := pt.initialDelay
	if pt.called {
		d = pt.period
	}

	pt.called = true
	return t.Add(d)
}
