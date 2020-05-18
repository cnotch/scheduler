// Copyright (c) 2019,CAO HONGJU. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package scheduler

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/cnotch/scheduler/cron"
)

const (
	// MinPeriod minimum period or delay
	MinPeriod = time.Microsecond
)

// PanicHandler is to handle panic caused by an asynchronous job.
type PanicHandler func(r interface{})

// IndPeriod executes f for a fixed period.
// If the execution time of f exceeds the period, there will
// be multiple instances of f running at the same time.
func IndPeriod(initialDelay, period time.Duration, f func(), ph PanicHandler) context.CancelFunc {
	if period < MinPeriod {
		panic("preiod must not be less than 1μs")
	}

	ctx, cancel := context.WithCancel(context.Background())

	safeRun := safeWrap(f, ph)

	go func() {
		if initialDelay <= 0 {
			initialDelay = 0
		}

		// first delay execute
		{
			timer := time.NewTimer(initialDelay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
				go safeRun()
			}
		}

		// Execute by period
		timer := time.NewTicker(period)
		for {
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
				go safeRun()
			}
		}
	}()

	return cancel
}

// IndDelay executes f at a fixed delay.
// The next execution time is delayed 'delay' after the last execution.
// Unlike IndPeriod, it never has multiple instances of f running at the same time.
func IndDelay(initialDelay, delay time.Duration, f func(), ph PanicHandler) context.CancelFunc {
	if delay < MinPeriod {
		panic("delay must not be less than 1μs")
	}

	ctx, cancel := context.WithCancel(context.Background())
	safeRun := safeWrap(f, ph)

	go func() {
		d := initialDelay

		for {
			if d < 0 {
				d = 0
			}

			timer := time.NewTimer(d)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
				// execute on the current go routine directly
				safeRun()
				// set the next delay
				d = delay
			}
		}
	}()

	return cancel
}

// IndCron Execute f according to a cron expression
func IndCron(expression string, f func(), ph PanicHandler) (cancel context.CancelFunc, err error) {
	var cronExp *cron.Expression
	cronExp, err = cron.Parse(expression)
	if err != nil {
		return
	}
	cancel = IndSchedule(cronExp, f, ph)
	return
}

// IndSchedule executes f according to schedule
func IndSchedule(schedule Schedule, f func(), ph PanicHandler) context.CancelFunc {
	ctx, cancel := context.WithCancel(context.Background())
	safeRun := safeWrap(f, ph)

	go func() {
		next := time.Now()

		for !next.IsZero() { // there is no more time, exit
			next = schedule.Next(next)
			d := next.Sub(time.Now())

			// If the execution time has expired, execute immediately
			if d < 0 {
				go safeRun()
				continue
			}

			timer := time.NewTimer(d)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
				go safeRun()
			}
		}
	}()

	return cancel
}

func safeWrap(f func(), ph PanicHandler) func() {
	return func() {
		defer func() {
			if r := recover(); r != nil {
				if ph != nil {
					ph(r)
				} else {
					fmt.Fprintf(os.Stderr, "panic: %+v", r)
				}
			}
		}()
		f()
	}
}
