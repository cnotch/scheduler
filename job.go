// Copyright (c) 2019,CAO HONGJU. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package scheduler

import (
	"sync/atomic"
	"time"
	"unsafe"
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
	// heap fields
	index int // index of the job in the heap
	// immutable fields of the job
	tag      interface{} // job tag, application provide
	schelule Schedule
	job      Job
	remove   chan *ManagedJob
	postTime time.Time

	// runtime fields
	next     time.Time // next trigger time
	prevTime lockedTime
	nextTime lockedTime
	// TODO: more...
}

// Cancel cancel the scheduled job.
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

// PostTime returns the time the job was posted to the scheduler
func (mjob *ManagedJob) PostTime() time.Time {
	return mjob.postTime
}

// PrevTime returns the prev execution time of the job.
func (mjob *ManagedJob) PrevTime() time.Time {
	return mjob.prevTime.get().In(mjob.postTime.Location())
}

// NextTime returns the next execution time of the job.
func (mjob *ManagedJob) NextTime() time.Time {
	return mjob.nextTime.get().In(mjob.postTime.Location())
}

func (mjob *ManagedJob) setNext(next time.Time) {
	mjob.prevTime.set(mjob.next)
	mjob.next = next
	mjob.nextTime.set(next)
}

type lockedTime struct {
	wall uint64
	ext  int64
}

func (lt *lockedTime) set(t time.Time) {
	temp := (*lockedTime)(unsafe.Pointer(&t))
	atomic.StoreUint64(&lt.wall, temp.wall)
	atomic.StoreInt64(&lt.ext, temp.ext)
}
func (lt *lockedTime) get() (t time.Time) {
	temp := (*lockedTime)(unsafe.Pointer(&t))
	temp.wall = atomic.LoadUint64(&lt.wall)
	temp.ext = atomic.LoadInt64(&lt.ext)
	return
}
