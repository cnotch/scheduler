// Copyright (c) 2019,CAO HONGJU. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package scheduler

import (
	"sync/atomic"
	"time"
)

// Job represent a 'job' to be performed.
type Job interface {
	// Run called by the Scheduler When the Schedule associated with the Job is triggered.
	Run()
}

// JobFunc is an adapter to allow the use of ordinary functions as the Job interface.
type JobFunc func()

// Run called by the Scheduler When the Schedule associated with the Job is triggered.
func (jf JobFunc) Run() {
	jf()
}

// ManagedJob represent the job managed by the scheduler.
type ManagedJob struct {
	// immutable fields of the job
	tag      interface{} // job tag, application provide
	schelule Schedule
	job      Job
	remove   chan *ManagedJob
	loc      *time.Location
	// heap fields
	index int // index of the job in the heap
	// runtime fields
	next     time.Time // next trigger time
	nextNano int64
	// TODO: more...
}

// Cancel 从计划任务中取消
func (mjob *ManagedJob) Cancel() {
	defer func() {
		if r := recover(); r != nil {
			// when mjob.remove closed
		}
	}()

	mjob.remove <- mjob
}

// Tag returns the tag of the job.
func (mjob *ManagedJob) Tag() interface{} {
	return mjob.tag
}

// Schelule returns the schedule of the job.
func (mjob *ManagedJob) Schelule() Schedule {
	return mjob.schelule
}

// Job return the executive job  of the job.
func (mjob *ManagedJob) Job() Job {
	return mjob.job
}

// Next returns the next execution time of the job.
func (mjob *ManagedJob) Next() time.Time {
	return time.Unix(0, atomic.LoadInt64(&mjob.nextNano)).In(mjob.loc)
}

func (mjob *ManagedJob) setNext(next time.Time) {
	mjob.next = next.In(mjob.loc)
	atomic.StoreInt64(&mjob.nextNano, next.UnixNano())
}
