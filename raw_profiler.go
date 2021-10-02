//go:build linux
// +build linux

package perf

import (
	"go.uber.org/multierr"
	"golang.org/x/sys/unix"
)

const (
	INT_MISC_RECOVERY_CYCLES_ANY  = 0x20010d
	CYCLE_ACTIVITY_STALLS_L2_MISS = 0x50005a3
	CYCLE_ACTIVITY_STALLS_L3_MISS = 0x60006a3
	CYCLE_ACTIVITY_STALLS_L1D_MISS = 0xc000ca3
	CYCLE_ACTIVITY_STALLS_MEM_ANY  = 0x140014a3
)

// Todo: Read this from a json file
type PerfRawEvent struct {
    Key    string `json:"key"`
	Type   uint32  `json:"type"`
    Config uint64 `json:"config"`
	Value  uint64 `json:"value"`
}

var events = []PerfRawEvent{
	{Key: "INT_MISC.RECOVERY_CYCLES_ANY",  Type: unix.PERF_TYPE_RAW, Config: INT_MISC_RECOVERY_CYCLES_ANY},
	{Key: "CYCLE_ACTIVITY.STALLS_L2_MISS", Type: unix.PERF_TYPE_RAW, Config: CYCLE_ACTIVITY_STALLS_L2_MISS},
	{Key: "CYCLE_ACTIVITY.STALLS_L3_MISS", Type: unix.PERF_TYPE_RAW, Config: CYCLE_ACTIVITY_STALLS_L3_MISS},
	{Key: "CYCLE_ACTIVITY.STALLS_L1D_MISS",Type: unix.PERF_TYPE_RAW, Config: CYCLE_ACTIVITY_STALLS_L1D_MISS},
	{Key: "CYCLE_ACTIVITY.STALLS_MEM_ANY", Type: unix.PERF_TYPE_RAW, Config: CYCLE_ACTIVITY_STALLS_MEM_ANY},
}
// 

type rawProfiler struct {
	// map of perf counter type to file descriptor
	profilers map[int]Profiler
}

// NewRawProfiler returns a new raw profiler.
func NewRawProfiler(pid, cpu int, opts ...int) (RawProfiler, error) {
	profilers := map[int]Profiler{}
	var e error

	for _,perfRawEvent := range events {
        tmpProfiler, err := NewProfiler(
			perfRawEvent.Type,
			perfRawEvent.Config,
			pid,
			cpu,
			opts...,
		) 
	
		if err != nil {
			e = multierr.Append(e, err)
		} else {
			profilers[int(perfRawEvent.Config)] = tmpProfiler
		}
    }

	return &rawProfiler{
		profilers: profilers,
	}, e
}

// Start is used to start the RawProfiler, it will return an error if no
// profilers are configured.
func (p *rawProfiler) Start() error {
	if len(p.profilers) == 0 {
		return ErrNoProfiler
	}
	var err error
	for _, profiler := range p.profilers {
		err = multierr.Append(err, profiler.Start())
	}
	return err
}

// Reset is used to reset the RawProfiler.
func (p *rawProfiler) Reset() error {
	var err error
	for _, profiler := range p.profilers {
		err = multierr.Append(err, profiler.Reset())
	}
	return err
}

// Stop is used to reset the RawProfiler.
func (p *rawProfiler) Stop() error {
	var err error
	for _, profiler := range p.profilers {
		err = multierr.Append(err, profiler.Stop())
	}
	return err
}

// Close is used to reset the RawProfiler.
func (p *rawProfiler) Close() error {
	var err error
	for _, profiler := range p.profilers {
		err = multierr.Append(err, profiler.Close())
	}
	return err
}

// Profile is used to read the RawProfiler RawProfile it returns an
// error only if all profiles fail.
func (p *rawProfiler) Profile() (*RawProfile, error) {
	var err error
	rawProfile := &RawProfile{}
	for profilerType, profiler := range p.profilers {
		profileVal, err2 := profiler.Profile()
		err = multierr.Append(err, err2)
		if err2 == nil {
			if rawProfile.TimeEnabled == nil {
				rawProfile.TimeEnabled = &profileVal.TimeEnabled
			}
			if rawProfile.TimeRunning == nil {
				rawProfile.TimeRunning = &profileVal.TimeRunning
			}
			switch {
			// L1 data
			case (profilerType ^ INT_MISC_RECOVERY_CYCLES_ANY) == 0:
				rawProfile.INT_MISC_RECOVERY_CYCLES_ANY = &profileVal.Value
			case (profilerType ^ CYCLE_ACTIVITY_STALLS_L2_MISS) == 0:
				rawProfile.CYCLE_ACTIVITY_STALLS_L2_MISS = &profileVal.Value
			case (profilerType ^ CYCLE_ACTIVITY_STALLS_L3_MISS) == 0:
				rawProfile.CYCLE_ACTIVITY_STALLS_L3_MISS = &profileVal.Value
			case (profilerType ^ CYCLE_ACTIVITY_STALLS_L1D_MISS) == 0:
				rawProfile.CYCLE_ACTIVITY_STALLS_L1D_MISS = &profileVal.Value
			case (profilerType ^ CYCLE_ACTIVITY_STALLS_MEM_ANY) == 0:
				rawProfile.CYCLE_ACTIVITY_STALLS_MEM_ANY = &profileVal.Value
			}
		}
	}
	if len(multierr.Errors(err)) == len(p.profilers) {
		return nil, err
	}

	return rawProfile, nil
}
