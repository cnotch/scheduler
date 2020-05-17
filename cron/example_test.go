// Copyright (c) 2019,CAO HONGJU. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
//
/*!
 * Copyright 2013 Raymond Hill
 *
 * Project: github.com/gorhill/example_test.go
 * File: example_test.go
 * Version: 1.0
 * License: GPL v3 see <https://www.gnu.org/licenses/gpl.html>
 *
 */

package cron

import (
	"fmt"
	"time"
)

// ExampleMustParse
func ExampleMustParse() {
	t := time.Date(2013, time.August, 31, 0, 0, 0, 0, time.UTC)
	expr := MustParse("0 0 29 2 *")

	for i := 0; i < 5; i++ {
		t = expr.Next(t)
		fmt.Println(t.Format(time.RFC1123))
	}
	// Output:
	// Mon, 29 Feb 2016 00:00:00 UTC
	// Sat, 29 Feb 2020 00:00:00 UTC
	// Thu, 29 Feb 2024 00:00:00 UTC
	// Tue, 29 Feb 2028 00:00:00 UTC
	// Sun, 29 Feb 2032 00:00:00 UTC
}
