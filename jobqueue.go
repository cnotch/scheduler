// Copyright (c) 2019,CAO HONGJU. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package scheduler

import (
	"container/heap"
	"time"
)

// A jobQueue implements heap.Interface and holds ManagedJobs.
type jobQueue []*ManagedJob

func (jobs jobQueue) Len() int { return len(jobs) }

func (jobs jobQueue) Less(i, j int) bool {
	return jobs[i].next.Before(jobs[j].next)
}

func (jobs jobQueue) Swap(i, j int) {
	jobs[i], jobs[j] = jobs[j], jobs[i]
	jobs[i].index = i
	jobs[j].index = j
}

func (jobs *jobQueue) Push(x interface{}) {
	n := len(*jobs)
	job := x.(*ManagedJob)
	job.index = n
	*jobs = append(*jobs, job)
}

func (jobs *jobQueue) Pop() interface{} {
	old := *jobs
	n := len(old)
	job := old[n-1]
	old[n-1] = nil // avoid memory leak
	job.index = -1 // for safety
	*jobs = old[0 : n-1]
	return job
}

func (jobs *jobQueue) updateNext(job *ManagedJob, next time.Time) {
	job.next = next
	heap.Fix(jobs, job.index)
}
