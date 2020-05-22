// Copyright (c) 2019,CAO HONGJU. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package scheduler

import (
	"context"
	"time"
)

// An Option configures a Scheduler.
type Option interface {
	apply(*Scheduler)
}

// optionFunc wraps a func so it satisfies the Option interface.
type optionFunc func(*Scheduler)

func (f optionFunc) apply(s *Scheduler) {
	f(s)
}

// WithContext configures the context of the Scheduler.
func WithContext(ctx context.Context) Option {
	return optionFunc(func(s *Scheduler) {
		s.ctx, s.cancel = context.WithCancel(ctx)
	})
}

// WithLocation configures the location of the Scheduler.
func WithLocation(location *time.Location) Option {
	return optionFunc(func(s *Scheduler) {
		s.loc = location
	})
}

// WithPanicHandler configures the panic exception handler.
func WithPanicHandler(panicHandler PanicHandler) Option {
	return optionFunc(func(s *Scheduler) {
		if panicHandler == nil {
			return
		}
		s.panicHandler.Store(panicHandler)
	})
}
