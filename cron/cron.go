// Copyright (c) 2019,CAO HONGJU. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
//
/*!
 * Copyright 2013 Raymond Hill
 *
 * Project: github.com/gorhill/cronexpr
 * File: cronexpr.go
 * Version: 1.0
 * License: pick the one which suits you :
 *   GPL v3 see <https://www.gnu.org/licenses/gpl.html>
 *   APL v2 see <http://www.apache.org/licenses/LICENSE-2.0>
 *
 */

package cron

import (
	"math"
	"math/bits"
	"time"
)

const (
	notFountIdx        = 64
	startBit    uint64 = 1 << 63
	secondsMask uint64 = 0xfffffffffffffff0
	minutesMask uint64 = 0xfffffffffffffff0
	hoursMask   uint64 = 0xffffff0000000000
	daysMask    uint64 = 0x7fffffff00000000
	monthsMask  uint64 = 0x7ff8000000000000
	weeksMask   uint64 = 0x7ffffffff0000000
)

// A Expression represents a specific cron time expression.
type Expression struct {
	expression         string    // raw expression string
	seconds            uint64    // 0~59 bit
	minutes            uint64    // 0~59 bit
	hours              uint64    // 0~23 bit
	daysOfMonth        uint64    // 1~31 bit
	workdaysOfMonth    uint64    // 1~31 bit
	lastDayOfMonth     bool      // L Flag
	lastWorkdayOfMonth bool      // LW Flag
	months             uint64    // 1~12 bit
	daysOfWeek         uint64    // 1~35 bit(5 weeks)
	ithWeekdaysOfWeek  uint64    // 1~35 bit(# sections)
	lastWeekdaysOfWeek uint64    // 1~35 bit(L sections)
	years              [3]uint64 // 0~128 bit
}

// Next returns the closest time instant immediately following `fromTime` which
// matches the cron expression `expr`.
//
// The `time.Location` of the returned time instant is the same as that of
// `fromTime`.
//
// The zero value of time.Time is returned if no matching time instant exists
// or if a `fromTime` is itself a zero value.
func (expr *Expression) Next(fromTime time.Time) time.Time {
	// Special case
	if fromTime.IsZero() {
		return fromTime
	}

	// Since expr.nextSecond()-expr.nextMonth() expects that the
	// supplied time stamp is a perfect match to the underlying cron
	// expression, and since this function is an entry point where `fromTime`
	// does not necessarily matches the underlying cron expression,
	// we first need to ensure supplied time stamp matches
	// the cron expression. If not, this means the supplied time
	// stamp falls in between matching time stamps, thus we move
	// to closest future matching immediately upon encountering a mismatching
	// time stamp.

	// year
	v := fromTime.Year()
	year := expr.matchYear(v)
	if year == 0 {
		return time.Time{}
	}
	if v != year {
		return expr.nextYear(fromTime)
	}

	// month
	v = int(fromTime.Month())
	i := matchField(expr.months, monthsMask, v)
	if i == notFountIdx {
		return expr.nextYear(fromTime)
	}
	if v != i {
		return expr.nextMonth(fromTime)
	}

	actualDaysOfMonth := expr.calculateActualDaysOfMonth(fromTime.Year(), int(fromTime.Month()), fromTime.Location())
	if actualDaysOfMonth == 0 {
		return expr.nextMonth(fromTime)
	}

	// day of month
	v = fromTime.Day()
	i = matchField(actualDaysOfMonth, daysMask, v)
	if i == notFountIdx {
		return expr.nextMonth(fromTime)
	}
	if v != i {
		return expr.nextDayOfMonth(fromTime, actualDaysOfMonth)
	}

	// hour
	v = fromTime.Hour()
	i = matchField(expr.hours, hoursMask, v)
	if i == notFountIdx {
		return expr.nextDayOfMonth(fromTime, actualDaysOfMonth)
	}
	if v != i {
		return expr.nextHour(fromTime, actualDaysOfMonth)
	}

	// minute
	v = fromTime.Minute()
	i = matchField(expr.minutes, minutesMask, v)
	if i == notFountIdx {
		return expr.nextHour(fromTime, actualDaysOfMonth)
	}
	if v != i {
		return expr.nextMinute(fromTime, actualDaysOfMonth)
	}

	// second
	v = fromTime.Second()
	i = matchField(expr.seconds, secondsMask, v)
	if i == notFountIdx {
		return expr.nextMinute(fromTime, actualDaysOfMonth)
	}

	// If we reach this point, there is nothing better to do
	// than to move to the next second
	return expr.nextSecond(fromTime, actualDaysOfMonth)
}

func (expr *Expression) matchYear(year int) int {
	if year > 2099 {
		return 0
	}
	if year < 1970 {
		year = 1970
	}
	idx := year - 1970

	for i, bit := idx>>6, idx&0x3f; i < 3; i++ {
		found := matchField(expr.years[i], math.MaxUint64, bit)
		if found != notFountIdx {
			return i<<6 + found + 1970
		}
		bit = 0
	}
	return 0
}

func matchField(v uint64, mask uint64, i int) int {
	return 64 - bits.Len64(v&((mask<<i)>>i))
}
func minValue(v uint64) int {
	return 64 - bits.Len64(v)
}

func (expr *Expression) nextYear(t time.Time) time.Time {
	// Find index at which item in list is greater or equal to
	// candidate year
	year := expr.matchYear(t.Year() + 1)
	if year == 0 {
		return time.Time{}
	}
	// Year changed, need to recalculate actual days of month
	actualDaysOfMonth := expr.calculateActualDaysOfMonth(year, minValue(expr.months), t.Location())
	if actualDaysOfMonth == 0 {
		return expr.nextMonth(time.Date(
			year,
			time.Month(minValue(expr.months)),
			1,
			minValue(expr.hours),
			minValue(expr.minutes),
			minValue(expr.seconds),
			0,
			t.Location()))
	}
	return time.Date(
		year,
		time.Month(minValue(expr.months)),
		minValue(actualDaysOfMonth),
		minValue(expr.hours),
		minValue(expr.minutes),
		minValue(expr.seconds),
		0,
		t.Location())
}

func (expr *Expression) nextMonth(t time.Time) time.Time {
	// Find index at which item in list is greater or equal to
	// candidate month
	i := matchField(expr.months, monthsMask, int(t.Month())+1)
	if i == notFountIdx {
		return expr.nextYear(t)
	}
	// Month changed, need to recalculate actual days of month
	actualDaysOfMonth := expr.calculateActualDaysOfMonth(t.Year(), i, t.Location())
	if actualDaysOfMonth == 0 {
		return expr.nextMonth(time.Date(
			t.Year(),
			time.Month(i),
			1,
			minValue(expr.hours),
			minValue(expr.minutes),
			minValue(expr.seconds),
			0,
			t.Location()))
	}

	return time.Date(
		t.Year(),
		time.Month(i),
		minValue(actualDaysOfMonth),
		minValue(expr.hours),
		minValue(expr.minutes),
		minValue(expr.seconds),
		0,
		t.Location())
}

func (expr *Expression) nextDayOfMonth(t time.Time, actualDaysOfMonth uint64) time.Time {
	// Find index at which item in list is greater or equal to
	// candidate day of month
	i := matchField(actualDaysOfMonth, daysMask, t.Day()+1)
	if i == notFountIdx {
		return expr.nextMonth(t)
	}

	return time.Date(
		t.Year(),
		t.Month(),
		i,
		minValue(expr.hours),
		minValue(expr.minutes),
		minValue(expr.seconds),
		0,
		t.Location())
}

func (expr *Expression) nextHour(t time.Time, actualDaysOfMonth uint64) time.Time {
	// Find index at which item in list is greater or equal to
	// candidate hour
	i := matchField(expr.hours, hoursMask, t.Hour()+1)
	if i == notFountIdx {
		return expr.nextDayOfMonth(t, actualDaysOfMonth)
	}

	return time.Date(
		t.Year(),
		t.Month(),
		t.Day(),
		i,
		minValue(expr.minutes),
		minValue(expr.seconds),
		0,
		t.Location())
}

func (expr *Expression) nextMinute(t time.Time, actualDaysOfMonth uint64) time.Time {
	// Find index at which item in list is greater or equal to
	// candidate minute
	i := matchField(expr.minutes, minutesMask, t.Minute()+1)
	if i == notFountIdx {
		return expr.nextHour(t, actualDaysOfMonth)
	}

	return time.Date(
		t.Year(),
		t.Month(),
		t.Day(),
		t.Hour(),
		i,
		minValue(expr.seconds),
		0,
		t.Location())
}

func (expr *Expression) nextSecond(t time.Time, actualDaysOfMonth uint64) time.Time {
	// nextSecond() assumes all other fields are exactly matched
	// to the cron expression

	// Find index at which item in list is greater or equal to
	// candidate second
	i := matchField(expr.seconds, secondsMask, t.Second()+1)
	if i == notFountIdx {
		return expr.nextMinute(t, actualDaysOfMonth)
	}

	return time.Date(
		t.Year(),
		t.Month(),
		t.Day(),
		t.Hour(),
		t.Minute(),
		i,
		0,
		t.Location())
}

func (expr *Expression) calculateActualDaysOfMonth(year, month int, loc *time.Location) uint64 {
	firstDayOfMonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, loc)
	lastDayOfMonth := firstDayOfMonth.AddDate(0, 1, -1)

	lastDay := lastDayOfMonth.Day()
	// remove bits over lastDay
	thisMonthsMask := (daysMask >> (63 - lastDay)) << (63 - lastDay)

	// As per crontab man page (http://linux.die.net/man/5/crontab#):
	//  "The day of a command's execution can be specified by two
	//  "fields - day of month, and day of week. If both fields are
	//  "restricted (ie, aren't *), the command will be run when
	//  "either field matches the current time"

	// If both fields are not restricted, all days of the month are a hit
	if expr.daysOfMonth == daysMask && expr.daysOfWeek == weeksMask {
		return thisMonthsMask
	}

	lastWeekday := lastDayOfMonth.Weekday()
	firstWeekday := firstDayOfMonth.Weekday()
	actualDaysOfMonth := uint64(0)
	// day-of-month != `*`
	if expr.daysOfMonth != daysMask {
		// Days of month
		actualDaysOfMonth |= expr.daysOfMonth

		// Last day of month(L Flag)
		if expr.lastDayOfMonth {
			actualDaysOfMonth |= startBit >> lastDay
		}
		// Last work day of month(LW Flag)
		if expr.lastWorkdayOfMonth {
			workday := lastWorkdayOfMonth(lastDay, lastDayOfMonth.Weekday())
			actualDaysOfMonth |= startBit >> workday
		}
		// Work days of month ({Number}W)
		// As per Wikipedia: month boundaries are not crossed.
		workdaysOfMonth := expr.workdaysOfMonth & thisMonthsMask
		if workdaysOfMonth > 0 {
			start := 64 - bits.Len64(workdaysOfMonth)
			end := 63 - bits.TrailingZeros64(workdaysOfMonth)
			if start == 1 {
				workday := firstWorkdayOfMonth(firstWeekday)
				actualDaysOfMonth |= startBit >> workday
				start++
			}
			for v := start; v <= end && v < lastDay; v++ {
				if workdaysOfMonth&(startBit>>v) != 0 {
					workday := midWorkdayOfMonth(v, time.Weekday(int(firstWeekday)+v-1)%7)
					actualDaysOfMonth |= startBit >> workday
				}
			}
			if end == lastDay {
				workday := lastWorkdayOfMonth(lastDay, lastWeekday)
				actualDaysOfMonth |= startBit >> workday
			}
		}
	}

	// day-of-week != `*`
	if expr.daysOfWeek != weeksMask {
		// days of week
		// expr.daysOfWeek << to set bit 1 is the frist day of month
		actualDaysOfMonth |= expr.daysOfWeek << int(firstWeekday)

		// days of week of specific week in the month(4#2)
		// expr.specificWeekDaysOfWeek << to set bit 1 is the frist day of month
		actualDaysOfMonth |= expr.ithWeekdaysOfWeek << int(firstWeekday)

		// Last days of week of the month({Weekday}L)
		// expr.lastWeekDaysOfWeek << to set bit 1 is the frist day of month
		lastWeekdays := expr.lastWeekdaysOfWeek << int(firstWeekday)
		// keep it for the last week
		lastWeekdays = (lastWeekdays << (lastDay - 7)) >> (lastDay - 7)
		actualDaysOfMonth |= lastWeekdays
	}

	// remove bits over lastDay
	return actualDaysOfMonth & thisMonthsMask
}

func lastWorkdayOfMonth(lastDay int, lastWeekday time.Weekday) int {
	switch lastWeekday {
	case time.Saturday:
		return lastDay - 1
	case time.Sunday:
		return lastDay - 2
	default:
		return lastDay
	}
}

func firstWorkdayOfMonth(firstWeekday time.Weekday) int {
	switch firstWeekday {
	case time.Saturday:
		return 3
	case time.Sunday:
		return 2
	default:
		return 1
	}
}

func midWorkdayOfMonth(midDay int, weekday time.Weekday) int {
	switch weekday {
	case time.Saturday:
		return midDay - 1
	case time.Sunday:
		return midDay + 1
	default:
		return midDay
	}
}
