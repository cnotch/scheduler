// Copyright (c) 2019,CAO HONGJU. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package cron

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

var (
	allYears = [3]uint64{math.MaxUint64, math.MaxUint64, math.MaxUint64}
)

// MustParse returns a new Expression pointer.
// It expects a well-formed cron expression.
// If a malformed cron expression is supplied, it will `panic`.
func MustParse(spec string) *Expression {
	expr, err := Parse(spec)
	if err != nil {
		panic(err)
	}
	return expr
}

// Parse returns a new Expression pointer.
// An error is returned if a malformed cron expression is supplied.
func Parse(spec string) (*Expression, error) {
	cron := strings.TrimSpace(spec)
	if len(cron) == 0 {
		return nil, fmt.Errorf("empty spec string")
	}

	// Handle named cron expression
	if strings.HasPrefix(cron, "@") {
		return parseNamedExpression(cron)
	}

	// Handle normalize cron expression
	expr := &Expression{expression: spec}
	fields := strings.Split(cron, " ")
	// remove empty fields
	for i := len(fields) - 1; i >= 0; i-- {
		if len(fields[i]) == 0 {
			copy(fields[i:], fields[i+1:])
			fields = fields[:len(fields)-1]
		}
	}

	fieldCount := len(fields)
	if fieldCount < 5 {
		return nil, fmt.Errorf("missing field(s)")
	}

	field := 0
	parser := 0
	// second field (optional)
	if fieldCount == 5 {
		expr.seconds = 1 << 63 // 0 second
		parser++               // set minute parser to the first
	}

	for field < fieldCount && parser < len(fieldParsers) {
		if err := fieldParsers[parser].parse(expr, fields[field]); err != nil {
			return nil, err
		}
		field++
		parser++
	}

	// padding years to all values
	if fieldCount < 7 {
		expr.years = allYears
	}

	// special handling for day of week
	// 7->0
	if expr.daysOfWeek&(1<<(63-7)) != 0 {
		expr.daysOfWeek |= 1 << 63
	}
	if expr.lastWeekdaysOfWeek&(1<<(63-7)) != 0 {
		expr.lastWeekdaysOfWeek |= 1 << 63
	}

	// expand to 5 week
	mask := uint64(0xfe00000000000000)
	daysOfWeek := expr.daysOfWeek & mask
	lastWeekdaysOfWeek := expr.lastWeekdaysOfWeek
	for i := 0; i < 35; i += 7 {
		expr.daysOfWeek |= daysOfWeek >> i
		expr.lastWeekdaysOfWeek |= lastWeekdaysOfWeek >> i
	}

	// sun to bit 1
	expr.daysOfWeek >>= 1
	expr.lastWeekdaysOfWeek >>= 1

	return expr, nil
}

func parseNamedExpression(spec string) (*Expression, error) {
	switch spec {
	case "@yearly", "@annually":
		return &Expression{
			expression:  spec, //0 0 0 1 1 * *
			seconds:     1 << 63,
			minutes:     1 << 63,
			hours:       1 << 63,
			daysOfMonth: 1 << 62,
			months:      1 << 62,
			daysOfWeek:  weeksMask,
			years:       allYears,
		}, nil
	case "@monthly":
		return &Expression{
			expression:  spec, // 0 0 0 1 * * *
			seconds:     1 << 63,
			minutes:     1 << 63,
			hours:       1 << 63,
			daysOfMonth: 1 << 62,
			months:      monthsMask,
			daysOfWeek:  weeksMask,
			years:       allYears,
		}, nil
	case "@weekly":
		return &Expression{
			expression:  spec, // 0 0 0 * * 0 *
			seconds:     1 << 63,
			minutes:     1 << 63,
			hours:       1 << 63,
			daysOfMonth: daysMask,
			months:      monthsMask,
			daysOfWeek:  genWeekdayBits([7]bool{0: true}),
			years:       allYears,
		}, nil
	case "@daily", "@midnight":
		return &Expression{
			expression:  spec, // 0 0 0 * * * *
			seconds:     1 << 63,
			minutes:     1 << 63,
			hours:       1 << 63,
			daysOfMonth: daysMask,
			months:      monthsMask,
			daysOfWeek:  weeksMask,
			years:       allYears,
		}, nil
	case "@hourly":
		return &Expression{
			expression:  spec, // 0 0 * * * * *
			seconds:     1 << 63,
			minutes:     1 << 63,
			hours:       hoursMask,
			daysOfMonth: daysMask,
			months:      monthsMask,
			daysOfWeek:  weeksMask,
			years:       allYears,
		}, nil
	}

	return nil, fmt.Errorf("unrecognized name of cron expression: %s", spec)
}

var fieldParsers = []fieldParser{
	{
		"second",
		func(expr *Expression, begin, end, step int) {
			for i := begin; i <= end; i += step {
				expr.seconds |= 1 << (63 - i)
			}
		},
		0, 59,
		atoi,
		nil,
	},
	{
		"minute",
		func(expr *Expression, begin, end, step int) {
			for i := begin; i <= end; i += step {
				expr.minutes |= 1 << (63 - i)
			}
		},
		0, 59,
		atoi,
		nil,
	},
	{
		"hour",
		func(expr *Expression, begin, end, step int) {
			for i := begin; i <= end; i += step {
				expr.hours |= 1 << (63 - i)
			}
		},
		0, 23,
		atoi,
		nil,
	},
	{
		"day of month",
		func(expr *Expression, begin, end, step int) {
			for i := begin; i <= end; i += step {
				expr.daysOfMonth |= 1 << (63 - i)
			}
		},
		1, 31,
		atoi,
		parseSpecDomEntry,
	},
	{
		"month",
		func(expr *Expression, begin, end, step int) {
			for i := begin; i <= end; i += step {
				expr.months |= 1 << (63 - i)
			}
		},
		1, 12,
		atomi,
		nil,
	},
	{
		"day of week",
		func(expr *Expression, begin, end, step int) {
			for i := begin; i <= end; i += step {
				expr.daysOfWeek |= 1 << (63 - i)
			}
		},
		0, 7,
		atowi,
		parseSpecDowEntry,
	},
	{
		"year",
		func(expr *Expression, begin, end, step int) {
			for i := begin - 1970; i <= end-1970; i += step {
				expr.years[i>>6] |= 1 << (63 - i&0x3f)
			}
		},
		1970, 2099,
		atoi,
		nil,
	},
}

const errPattern = "syntax error in %s field: '%s'"

type fieldParser struct {
	name            string
	populateTo      func(expr *Expression, begin, end, step int)
	min, max        int
	atoi            func(string) (int, bool)
	specEntryParser func(expr *Expression, entry string, atoi func(string) (int, bool)) bool
}

func (fp *fieldParser) parse(expr *Expression, field string) error {
	idx := strings.IndexByte(field, ',')
	if idx == -1 {
		return fp.parseEntry(expr, field)
	}

	entrys := strings.Split(field, ",")
	for _, entry := range entrys {
		err := fp.parseEntry(expr, entry)
		if err != nil {
			return err
		}
	}
	return nil
}

func (fp *fieldParser) parseStep(expr *Expression, entry string, step int) bool {
	if entry == "*" { // min-max
		fp.populateTo(expr, fp.min, fp.max, step)
		return true
	}

	n, ok := fp.atoi(entry)
	if ok { // n-max
		if !fp.isValid(n) {
			return false
		}
		fp.populateTo(expr, n, fp.max, step)
		return true
	}

	// standard begin-end
	idx := strings.IndexByte(entry, '-')
	begin, ok := fp.atoi(entry[:idx])
	if !ok || !fp.isValid(begin) {
		return false
	}
	end, ok := fp.atoi(entry[idx+1:])
	if !ok || !fp.isValid(end) {
		return false
	}
	fp.populateTo(expr, begin, end, step)
	return true
}

func (fp *fieldParser) parseEntry(expr *Expression, entry string) error {
	if entry == "*" {
		fp.populateTo(expr, fp.min, fp.max, 1)
		return nil
	}
	n, ok := fp.atoi(entry)
	if ok { // one value
		if !fp.isValid(n) {
			return fmt.Errorf(errPattern, fp.name, entry)
		}
		fp.populateTo(expr, n, n, 1)
		return nil
	}

	// step  /
	idx := strings.IndexByte(entry, '/')
	if idx != -1 {
		step, ok := fp.atoi(entry[idx+1:])
		if !ok || step < 1 || step > (fp.max-fp.min) {
			return fmt.Errorf(errPattern, fp.name, entry)
		}
		if !fp.parseStep(expr, entry[:idx], step) {
			return fmt.Errorf(errPattern, fp.name, entry)
		}
		return nil
	}

	// span
	idx = strings.IndexByte(entry, '-')
	if idx != -1 {
		if !fp.parseStep(expr, entry, 1) {
			return fmt.Errorf(errPattern, fp.name, entry)
		}
		return nil
	}
	if fp.specEntryParser == nil || !fp.specEntryParser(expr, entry, fp.atoi) {
		return fmt.Errorf(errPattern, fp.name, entry)
	}
	return nil
}

func (fp *fieldParser) isValid(n int) bool {
	return n >= fp.min && n <= fp.max
}

func parseSpecDomEntry(expr *Expression, entry string, atoi func(string) (int, bool)) bool {
	const min, max = int(1), int(31)
	if entry == "?" {
		expr.daysOfMonth |= daysMask
		return true
	}
	if entry == "LW" {
		expr.lastWorkdayOfMonth = true
		return true
	}
	if entry == "L" {
		expr.lastDayOfMonth = true
		return true
	}
	if strings.HasSuffix(entry, "W") {
		n, ok := atoi(entry[:len(entry)-1])
		if !ok || n < min || n > max {
			return false
		}
		expr.workdaysOfMonth |= 1 << (63 - n)
		return true
	}
	return false
}

func parseSpecDowEntry(expr *Expression, entry string, atoi func(string) (int, bool)) bool {
	const min, max = int(0), int(7)
	if entry == "?" {
		expr.daysOfWeek |= weeksMask << 1
		return true
	}
	if strings.HasSuffix(entry, "L") {
		n, ok := atoi(entry[:len(entry)-1])
		if !ok || n < min || n > max {
			return false
		}
		expr.lastWeekdaysOfWeek |= 1 << (63 - n)
		return true
	}

	idx := strings.IndexByte(entry, '#')
	if idx != -1 {
		weekday, ok := atowi(entry[:idx])
		if !ok || weekday < min || weekday > max {
			return false
		}
		ith, ok := atowi(entry[idx+1:])
		if !ok || ith < 1 || ith > 5 {
			return false
		}
		if weekday == 7 {
			weekday = 0
		}
		n := (ith-1)*7 + weekday
		expr.ithWeekdaysOfWeek |= 1 << (63 - n - 1) // sun is bit 1
		return true
	}
	return false
}

func genWeekdayBits(weekdays [7]bool) uint64 {
	var v uint64
	i := 1 // day start from 1
	for k := 0; k < 5; k++ {
		for _, b := range weekdays {
			if b {
				v |= 1 << (63 - i)
			}
			i++
		}
	}
	return v
}

func atoi(s string) (int, bool) {
	i, err := strconv.Atoi(s)
	return i, err == nil
}

func atowi(s string) (int, bool) {
	switch strings.ToLower(s) {
	case `0`, `sun`, `sunday`:
		return 0, true
	case `1`, `mon`, `monday`:
		return 1, true
	case `2`, `tue`, `tuesday`:
		return 2, true
	case `3`, `wed`, `wednesday`:
		return 3, true
	case `4`, `thu`, `thursday`:
		return 4, true
	case `5`, `fri`, `friday`:
		return 5, true
	case `6`, `sat`, `saturday`:
		return 6, true
	case `7`:
		return 7, true
	default:
		return 0, false
	}
}

func atomi(s string) (int, bool) {
	switch strings.ToLower(s) {
	case `1`, `jan`, `january`:
		return 1, true
	case `2`, `feb`, `february`:
		return 2, true
	case `3`, `mar`, `march`:
		return 3, true
	case `4`, `apr`, `april`:
		return 4, true
	case `5`, `may`:
		return 5, true
	case `6`, `jun`, `june`:
		return 6, true
	case `7`, `jul`, `july`:
		return 7, true
	case `8`, `aug`, `august`:
		return 8, true
	case `9`, `sep`, `september`:
		return 9, true
	case `10`, `oct`, `october`:
		return 10, true
	case `11`, `nov`, `november`:
		return 11, true
	case `12`, `dec`, `december`:
		return 12, true
	default:
		return 0, false
	}
}
