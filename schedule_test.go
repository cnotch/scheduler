// Copyright (c) 2018,TianJin Tomatox  Technology Ltd. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package scheduler

import (
	"fmt"
	"testing"
	"time"

	"github.com/cnotch/scheduler/cron"
	"github.com/stretchr/testify/assert"
)

type compsitetime struct {
	from     string
	expected bool
}
type compsitetest struct {
	compsite     func(l, r Schedule) Schedule
	op           string
	spec1, spec2 string
	layout       string
	times        []compsitetime
}

var compsitetests = []compsitetest{
	{
		Union, "∪",
		"0 0/6 * * * *", "0 0/15 * * * *",
		"Mon Jan 2 15:04 2006",
		[]compsitetime{
			{"Mon Jul 9 15:00 2012", true},
			{"Mon Jul 9 15:06 2012", true},
			{"Mon Jul 9 15:12 2012", true},
			{"Mon Jul 9 15:15 2012", true},
			{"Mon Jul 9 15:16 2012", false},
			{"Mon Jul 9 15:18 2012", true},
		},
	},
	{
		Minus, "-",
		"0 0/6 * * * *", "0 0/15 * * * *",
		"Mon Jan 2 15:04 2006",
		[]compsitetime{
			{"Mon Jul 9 15:00 2012", false},
			{"Mon Jul 9 15:06 2012", true},
			{"Mon Jul 9 15:12 2012", true},
			{"Mon Jul 9 15:15 2012", false},
			{"Mon Jul 9 15:16 2012", false},
			{"Mon Jul 9 15:18 2012", true},
		},
	},
	{
		Intersect, "∩",
		"0 0/6 * * * *", "0 0/15 * * * *",
		"Mon Jan 2 15:04 2006",
		[]compsitetime{
			{"Mon Jul 9 15:00 2012", true},
			{"Mon Jul 9 15:06 2012", false},
			{"Mon Jul 9 15:12 2012", false},
			{"Mon Jul 9 15:15 2012", false},
			{"Mon Jul 9 15:16 2012", false},
			{"Mon Jul 9 15:18 2012", false},
			{"Mon Jul 9 15:30 2012", true},
		},
	},
}

func TestCompsite(t *testing.T) {
	for _, test := range compsitetests {
		cron1 := cron.MustParse(test.spec1)
		cron2 := cron.MustParse(test.spec2)
		comp := test.compsite(cron1, cron2)

		for _, ctime := range test.times {
			from, _ := time.Parse(test.layout, ctime.from)
			from = from.Add(-1 * time.Second)
			next := comp.Next(from)
			nextstr := next.Format(test.layout)
			if ctime.expected {
				assert.True(t, ctime.from == nextstr, fmt.Sprintf("%s %s %s on %s",
					test.spec1, test.op, test.spec2, ctime.from))
			} else {
				assert.False(t, ctime.from == nextstr, fmt.Sprintf("%s %s %s on %s",
					test.spec1, test.op, test.spec2, ctime.from))
			}
		}
	}
}
