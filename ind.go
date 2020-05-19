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

// IndPeriod posts the function f which will execute the first time
// at the specified delay, followed by a fixed period.
// If the execution time of f exceeds the period, there will
// be multiple instances of f running at the same time.
func IndPeriod(initialDelay, period time.Duration, f func(), panicHandler func(r interface{})) context.CancelFunc {
	ctx, cancel := context.WithCancel(context.Background())

	safeRun := safeWrap(f, panicHandler)

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

// IndDelay posts the function f which will execute the first time
// at the specified delay, followed by a fixed dlay.
// The next execution time is delayed 'delay' after the last execution.
// Unlike IndPeriod, it never has multiple instances of f running at the same time.
func IndDelay(initialDelay, delay time.Duration, f func(), panicHandler func(r interface{})) context.CancelFunc {

	ctx, cancel := context.WithCancel(context.Background())
	safeRun := safeWrap(f, panicHandler)

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

// IndCron posts the function f which will execute according to a cron expression.
func IndCron(expression string, f func(), panicHandler func(r interface{})) (cancel context.CancelFunc, err error) {
	var cronExp *cron.Expression
	cronExp, err = cron.Parse(expression)
	if err != nil {
		return
	}
	cancel = IndSchedule(cronExp, f, panicHandler)
	return
}

// IndSchedule posts the function f which will execute according to the specified schedule.
func IndSchedule(schedule Schedule, f func(), panicHandler func(r interface{})) context.CancelFunc {
	ctx, cancel := context.WithCancel(context.Background())
	safeRun := safeWrap(f, panicHandler)

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

func safeWrap(f func(), panicHandler func(r interface{})) func() {
	return func() {
		defer func() {
			if r := recover(); r != nil {
				if panicHandler != nil {
					panicHandler(r)
				} else {
					fmt.Fprintf(os.Stderr, "panic: %+v", r)
				}
			}
		}()
		f()
	}
}
