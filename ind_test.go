// Copyright (c) 2018,TianJin Tomatox  Technology Ltd. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package scheduler

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIndPeriod(t *testing.T) {
	assert.NotPanics(t, func() {
		out := make(chan bool, 1)
		cancel := IndPeriod(0, time.Microsecond*10, func() {
			out <- true
		}, nil)

		<-out
		v := <-out
		assert.True(t, v)
		cancel()
	})
}

func TestIndDelay(t *testing.T) {
	assert.NotPanics(t, func() {
		out := make(chan bool, 1)
		cancel := IndDelay(0, time.Microsecond*10, func() {
			out <- true
		}, nil)

		<-out
		v := <-out
		assert.True(t, v)
		cancel()
	})
}

func TestIndDelayFirstActionPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		var panicRecv interface{}
		cancel := IndDelay(0, time.Microsecond*10, func() {
			panic("test")
		}, func(r interface{}) { panicRecv = r })

		<-time.After(time.Millisecond * 10)
		// <-time.After(time.Second)
		cancel()
		assert.Equal(t, "test", panicRecv)
	})
}

func TestIndDelayPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		var counter int32
		var panicRecv interface{}

		cancel := IndDelay(0, time.Microsecond*10, func() {
			atomic.AddInt32(&counter, 1)
			panic("test")
		}, func(r interface{}) { panicRecv = r })

		for atomic.LoadInt32(&counter) <= 2 {
		}

		cancel()
		assert.Equal(t, "test", panicRecv)
	})
}

func TestIndCron(t *testing.T) {
	assert.NotPanics(t, func() {
		var calls = 0
		cancel, _ := IndCron("* * * * * *", func() { calls++ }, nil)
		defer cancel()

		<-time.After(time.Second + 100*time.Nanosecond)
		if calls != 1 {
			t.Errorf("called %d times, expected 1\n", calls)
		}
	})
}
