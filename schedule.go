// Copyright (c) 2019,CAO HONGJU. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package scheduler

import "time"

// Schedule describes a job's duty cycle.
type Schedule interface {
	// Next returns the next activation time, later than the given time.
	// Next returns 0(Time.IsZero()) to indicate job termination.
	Next(time.Time) time.Time
}

// ScheduleFunc is an adapter to allow the use of ordinary functions as the Schedule interface.
type ScheduleFunc func(time.Time) time.Time

// Next returns the next activation time, later than the given time.
func (f ScheduleFunc) Next(t time.Time) time.Time {
	return f(t)
}

// Union returns the new schedule that union left schedule and right schedule(left ∪ right).
func Union(l, r Schedule) Schedule {
	return &union{
		l: l,
		r: r,
	}
}

type union struct {
	l Schedule
	r Schedule
}

func (us *union) Next(t time.Time) time.Time {
	t1 := us.l.Next(t)
	t2 := us.r.Next(t)
	if t1.Before(t2) && !t1.IsZero() {
		return t1
	}
	return t2
}

// Minus returns the new schedule that the left schedule minus the right schedule(l - r).
func Minus(l, r Schedule) Schedule {
	return &minus{
		l: l,
		r: r,
	}
}

type minus struct {
	l Schedule
	r Schedule
}

func (ms *minus) Next(t time.Time) time.Time {
	t1 := ms.l.Next(t)
	t2 := ms.r.Next(t)

	for {
		if t2.IsZero() {
			return t1
		}

		// t1 < t2
		if t1.Before(t2) {
			return t1
		}

		// t1 == t2, recalculated
		// the trigger condition is not valid
		if t1.Equal(t2) {
			t1 = ms.l.Next(t1)
			t2 = ms.r.Next(t2)
			continue
		}

		for t1.After(t2) { // t1 > t2
			t2 = ms.r.Next(t2)
		}
	}
}

// Intersect returns the intersection of left schedule and right schedule(l ∩ r).
func Intersect(l, r Schedule) Schedule {
	return &intersect{
		l: l,
		r: r,
	}
}

type intersect struct {
	l Schedule
	r Schedule
}

func (is *intersect) Next(t time.Time) time.Time {
	t1 := is.l.Next(t)
	t2 := is.r.Next(t)
	for {
		if t1.IsZero() || t2.IsZero() {
			return t1
		}

		if t1.Equal(t2) { // valid
			return t1
		}

		if t1.Before(t2) {
			t1 = is.l.Next(t1)
		} else {
			t2 = is.r.Next(t2)
		}
	}
}
