# scheduler
[scheduler](https://godoc.org/github.com/cnotch/scheduler) provides a lightning fast job scheduling library.

The ideas and design are based on the following projects:
+ [robfig/cron](https://github.com/robfig/cron)
+ [gorhill/cronexpr](https://github.com/gorhill/cronexpr)

## Features

- Best Performance: [Benchmarks speak for themselves](#benchmarks)
- Less memory allocation: [Benchmarks speak for themselves](#benchmarks)
- Support schedule union,minus and intersect operations.
- Provide an scheduled job that does not depend on Scheduler.

## Installing

1. Get package:

	```Shell
	go get -u github.com/cnotch/scheduler
	```

2. Import it in your code:

	```Go
	import "github.com/cnotch/scheduler"
	import "github.com/cnotch/scheduler/cron" // when use cron expressions directly
	```

## Usage

### Simple example

The following example is executed once per second(using Ind... function):

``` go
	var calls = 0
    cancel, _ := IndCron("* * * * * *", func() { calls++ }, nil)
    defer cancel()
    // other coder
```

The following example is executed for the first time at delay 1s, and then every minute(using Scheduler):

``` go
    var counter int32
	mj, _ := schd.PeriodFunc(time.Second, time.Minute, func() {
		atomic.AddInt32(&counter, 1)
    }, nil)
    // other code
    mj.Cancel()
```

### Handle panic example

The following example cancels the job when a panic occurs
``` go
    schd := scheduler.New(scheduler.WithPanicHandler(func(job *scheduler.ManagedJob, r interface{}) {
        // other handle code
		// If panic occurs, cancel the job;
		// Can also not cancel, continue to execute next time
		job.Cancel()
    }))

    //...
    var counter int32

	mj, _ := s.PeriodFunc(0, time.Millisecond*10, func() {
		atomic.AddInt32(&counter, 1)
		panic("test")
	}, nil)
```

### Compsite Schedule example

In China, many holidays are rescheduled so that people can have more time to travel. Take the May Day holiday in 2020 as an example, how do we get the work schedule?
Below is the time schedule that is triggered at 8:30am every working day. 

``` go
type whSchedule struct {
	times []time.Time
}

func (wh whSchedule) Next(t time.Time) time.Time {
	for _, lt := range wh.times {
		if lt.After(t) {
			return lt
		}
	}
	return time.Time{}
}

func ExampleCompsite() {
	t := time.Date(2020, 4, 25, 8, 30, 0, 0, time.Local)
	mon2fri := cron.MustParse("30 8 ? * 1-5")
	// The holiday (Saturday, Sunday) was transferred to a working day
	workday := whSchedule{[]time.Time{
		t.AddDate(0, 0, 1),  // 2020-04-26 08:30
		t.AddDate(0, 0, 14), // 2020-05-09 08:30
	}}

	// The working day (Monday - Friday) is changed into a holiday
	holiday := whSchedule{[]time.Time{
		t.AddDate(0, 0, 6), // 2020-05-01 08:30
		t.AddDate(0, 0, 9), // 2020-05-04 08:30
		t.AddDate(0, 0, 10),// 2020-05-05 08:30
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

```

### Design global Scheduler

Sometimes you just need to provide a global scheduler for your application. The following example shows how you can implement this requirement.

``` go
var (
	globalSchd = scheduler.New()
)

func init() {
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
			fmt.Fprintf(os.Stderr, "global scheduler received signal `%s`, exiting...", sig.String())
			globalSchd.ShutdownAndWait()
		}
	}
}

// AfterFunc posts the function f to the gloabl Scheduler.
func AfterFunc(delay time.Duration, f func(), tag interface{}) (*ManagedJob, error) {
	return globalSchd.AfterFunc(delay, f, tag)
}

// ... other function

```

## Benchmarks

gorhill/cronexpr
``` shell
BenchmarkParse-8   	   64550	     18287 ns/op	    5955 B/op	      79 allocs/op
BenchmarkNext-8    	  243950	      5676 ns/op	     517 B/op	      18 allocs/op
```

cnotch/scheduler
``` shell
BenchmarkParse-8   	 2024068	       575 ns/op	     249 B/op	       3 allocs/op
BenchmarkNext-8    	  425515	      2972 ns/op	       0 B/op	       0 allocs/op
```