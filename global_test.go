// Copyright (c) 2018,TianJin Tomatox  Technology Ltd. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package scheduler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAfterFunc(t *testing.T) {
	assert.NotPanics(t, func() {
		out := make(chan bool, 1)
		mj, _ := AfterFunc(time.Millisecond*10, func() {
			out <- true
		}, nil)
		v := <-out
		assert.True(t, v)
		mj.Cancel()
	})
}

func TestPeriodFunc(t *testing.T) {
	assert.NotPanics(t, func() {
		out := make(chan bool, 1)
		mjob, _ := PeriodFunc(0, time.Millisecond, func() {
			out <- true
		}, nil)

		<-out
		v := <-out
		assert.True(t, v)
		mjob.Cancel()
	})
}

func TestCronFunc(t *testing.T) {
	assert.NotPanics(t, func() {
		var calls = 0
		mjob, _ := CronFunc("* * * * * *", func() { calls++ }, nil)
		defer mjob.Cancel()

		<-time.After(time.Second + 100*time.Nanosecond)
		if calls != 1 {
			t.Errorf("called %d times, expected 1\n", calls)
		}
	})
}
