// Copyright (c) 2019,CAO HONGJU. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package scheduler

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	defaultSchd = New() // location = time.Local
)

func init() {
	// cleaning when system signal is received
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	go handleSignal(c)
}

func handleSignal(c <-chan os.Signal) {
	for sig := range c {
		switch sig {
		case syscall.SIGTERM:
			fallthrough
		case syscall.SIGINT:
			fmt.Fprintf(os.Stderr, "default scheduler received signal `%s`, exiting...\n", sig.String())
			defaultSchd.ShutdownAndWait()
			os.Exit(0)
		}
	}
}

// AfterFunc posts the function f to the default Scheduler.
// The function f will execute after specified delay only once,
// and then remove from the Scheduler.
func AfterFunc(delay time.Duration, f func(), tag interface{}) (*ManagedJob, error) {
	return defaultSchd.AfterFunc(delay, f, tag)
}

// After posts the job to the default Scheduler.
// The job will execute after specified delay only once,
// and then remove from the Scheduler.
func After(delay time.Duration, job Job, tag interface{}) (*ManagedJob, error) {
	return defaultSchd.After(delay, job, tag)
}

// PeriodFunc posts the function f to the default Scheduler.
// The function f will execute the first time at the specified delay,
// followed by a fixed period. If the execution time of f exceeds
// the period, there will be multiple instances of f running at the same time.
func PeriodFunc(initialDelay, period time.Duration, f func(), tag interface{}) (*ManagedJob, error) {
	return defaultSchd.PeriodFunc(initialDelay, period, f, tag)
}

// Period posts the job to the default Scheduler.
// The job will execute the first time at the specified delay,
// followed by a fixed period. If the execution time of job exceeds
// the period, there will be multiple instances of job running at the same time.
func Period(initialDelay, period time.Duration, job Job, tag interface{}) (*ManagedJob, error) {
	return defaultSchd.Period(initialDelay, period, job, tag)
}

// CronFunc posts the function f to the default Scheduler, and associate the given cron expression with it.
func CronFunc(cronExpr string, f func(), tag interface{}) (*ManagedJob, error) {
	return defaultSchd.CronFunc(cronExpr, f, tag)
}

// Cron posts the job to the default Scheduler, and associate the given cron expression with it.
func Cron(cronExpr string, job Job, tag interface{}) (*ManagedJob, error) {
	return defaultSchd.Cron(cronExpr, job, tag)
}

// PostFunc posts the function f to the default Scheduler, and associate the given schedule with it.
func PostFunc(schedule Schedule, f func(), tag interface{}) (*ManagedJob, error) {
	return defaultSchd.PostFunc(schedule, f, tag)
}

// Post posts the job to the default Scheduler, and associate the given schedule with it.
func Post(schedule Schedule, job Job, tag interface{}) (mjob *ManagedJob, err error) {
	return defaultSchd.Post(schedule, job, tag)
}

// Jobs returns the scheduled jobs of the global scheduler.
func Jobs() (jobs []*ManagedJob) {
	return defaultSchd.Jobs()
}

// Count returns jobs count of the global scheduler.
func Count() int {
	return defaultSchd.Count()
}

// Location returns the time zone location of the global scheduler.
func Location() *time.Location {
	return defaultSchd.Location()
}

// SetPanicHandler set the panic handler of the global scheduler.
func SetPanicHandler(panicHandler PanicHandler) {
	defaultSchd.SetPanicHandler(panicHandler)
}
