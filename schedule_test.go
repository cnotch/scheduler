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

type whSchedule struct {
	times []time.Time
}

func (wht whSchedule) Next(t time.Time) time.Time {
	for _, lt := range wht.times {
		if lt.After(t) {
			return lt
		}
	}
	return time.Time{}
}

// ExampleUnion
func ExampleUnion() {
	t := time.Date(2020, 4, 25, 8, 30, 0, 0, time.Local)
	mon2fri := cron.MustParse("30 8 ? * 1-5")
	// The holiday (Saturday, Sunday) was transferred to a working day
	workday := whSchedule{[]time.Time{
		t.AddDate(0, 0, 1),  // 2020-04-26 08:30
		t.AddDate(0, 0, 14), // 2020-05-09 08:30
	}}

	// The working day (Monday - Friday) is changed into a holiday
	holiday := whSchedule{[]time.Time{
		t.AddDate(0, 0, 6),  // 2020-05-01 08:30
		t.AddDate(0, 0, 9),  // 2020-05-04 08:30
		t.AddDate(0, 0, 10), // 2020-05-05 08:30
	}}
	workingSchedule := Union(Minus(mon2fri, holiday), workday)

	for i := 0; i < 12; i++ {
		t = workingSchedule.Next(t)
		fmt.Println(t.Format("2006-01-02 15:04:05"))
	}
	// Output:
	// 2020-04-26 08:30:00
	// 2020-04-27 08:30:00
	// 2020-04-28 08:30:00
	// 2020-04-29 08:30:00
	// 2020-04-30 08:30:00
	// 2020-05-06 08:30:00
	// 2020-05-07 08:30:00
	// 2020-05-08 08:30:00
	// 2020-05-09 08:30:00
	// 2020-05-11 08:30:00
	// 2020-05-12 08:30:00
	// 2020-05-13 08:30:00
}
