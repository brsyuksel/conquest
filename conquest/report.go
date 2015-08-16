package conquest

import (
	"conquest/utils"
	"fmt"
	"net/http"
	"os"
	"time"
)

const (
	REASON_TRANSACTION = 1 << iota
	REASON_RESPONSE
)

type Success struct {
	Path        string
	ElapsedTime time.Duration
}

type reason struct {
	Kind    uint8
	Error   error
	Request *http.Request
}
type Fail struct {
	Path        string
	ElapsedTime time.Duration
	Reason      *reason
}

type reportChannels struct {
	Fail    chan *Fail
	Success chan *Success
	Done    chan bool
}

type report struct {
	Hits        uint64
	Success     uint64
	Fails       uint64
	ElapsedTime time.Duration
	AverageTime time.Duration
	SlowestTime time.Duration
	FastestTime time.Duration
	Failed      map[string][]*reason
	Slowest     *Success
	Fastest     *Success
	C           *reportChannels
}

func write(r *report, f *os.File) {
STAT:
	for {
		select {
		case f := <-r.C.Fail:
			r.Hits++
			r.Fails++
			r.ElapsedTime += f.ElapsedTime

			if _, ok := r.Failed[f.Path]; !ok {
				r.Failed[f.Path] = []*reason{}
			}
			r.Failed[f.Path] = append(r.Failed[f.Path], f.Reason)

		case s := <-r.C.Success:
			r.Hits++
			r.Success++
			r.ElapsedTime += s.ElapsedTime

			if s.ElapsedTime > r.SlowestTime {
				r.SlowestTime = s.ElapsedTime
				r.Slowest = s
			}

			if r.FastestTime == 0 {
				r.FastestTime = s.ElapsedTime
				r.Fastest = s
			}
			if s.ElapsedTime < r.FastestTime {
				r.FastestTime = s.ElapsedTime
				r.Fastest = s
			}
		case <-r.C.Done:
			break STAT
		}
	}

	fmt.Fprintln(f, "Summary:")
	fmt.Fprintf(f, "Hits: %d Success: %d Fails: %d\n\n", r.Hits, r.Success, r.Fails)
	fmt.Fprintln(f, "Elapsed Time: ", utils.NS2MS(r.ElapsedTime.Nanoseconds()), " ms")
	fmt.Fprintln(f, "Average Time: ",
		utils.NS2MS(time.Duration(int64(r.ElapsedTime)/int64(r.Hits)).Nanoseconds()), " ms")
	fmt.Fprintln(f, "Slowest Time: ", utils.NS2MS(r.SlowestTime.Nanoseconds()), " ms")
	fmt.Fprintln(f, "Fastest Time: ", utils.NS2MS(r.FastestTime.Nanoseconds()), " ms")
	fmt.Fprintln(f, "")
	if r.Slowest != nil {
		fmt.Fprintln(f, "Slowest Transaction: ")
		fmt.Fprintln(f, "\tPath: ", r.Slowest.Path)
		fmt.Fprintln(f, "\tElapsed Time: ", utils.NS2MS(r.Slowest.ElapsedTime.Nanoseconds()), " ms")
	}

	if r.Fastest != nil {
		fmt.Fprintln(f, "Fastest Transaction: ")
		fmt.Fprintln(f, "\tPath: ", r.Fastest.Path)
		fmt.Fprintln(f, "\tElapsed Time: ", utils.NS2MS(r.Fastest.ElapsedTime.Nanoseconds()), " ms")
		fmt.Fprintln(f, "")
	}

	if len(r.Failed) > 0 {
		fmt.Fprintln(f, "Failed Transactions:")
		for path, reasons := range r.Failed {
			fmt.Fprintln(f, "\tPath: ", path)
			fmt.Fprintln(f, "\tReasons:")
			for _, r := range reasons {
				switch r.Kind {
				case REASON_RESPONSE:
					fmt.Fprintln(f, "\t\tResponse Error: ", r.Error.Error())
				case REASON_TRANSACTION:
					fmt.Fprintln(f, "\t\tTransaction Error: ", r.Error.Error())
				}
				/* FIXME: pretty print for failed request*/
				fmt.Fprintln(f, "\t\tRequest: ", r.Request)
			}
			fmt.Fprintln(f, "")
		}
	}
	r.C.Done <- true
}

func NewReporter(f *os.File) *report {
	r := &report{
		Failed: map[string][]*reason{},
		C: &reportChannels{
			Fail:    make(chan *Fail),
			Success: make(chan *Success),
			Done:    make(chan bool),
		},
	}

	go write(r, f)
	return r
}

func NewFail(k uint8, p string, err error, e time.Duration, r *http.Request) *Fail {
	f := &Fail{
		Path:        p,
		ElapsedTime: e,
		Reason: &reason{
			Kind:    k,
			Error:   err,
			Request: r,
		},
	}
	return f
}

func NewSuccess(p string, e time.Duration) *Success {
	return &Success{
		Path:        p,
		ElapsedTime: e,
	}
}
