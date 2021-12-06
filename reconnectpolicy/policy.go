package reconnectpolicy

import "time"

type Never struct{}

func (n Never) Retry() bool {
	return false
}

func (n Never) Timer() *time.Timer {
	return nil
}

type ConstTime struct {
	Duration time.Duration
}

func NewConstTime(d time.Duration) ConstTime {
	return ConstTime{Duration: d}
}

func (ct ConstTime) Retry() bool {
	return true
}

func (ct ConstTime) Timer() *time.Timer {
	return time.NewTimer(ct.Duration)
}

type ExponentTime struct {
	retry int

	InitDuration time.Duration
	MaxDuration  time.Duration
	MaxRetry     int
}

func (et *ExponentTime) Retry() bool {
	et.retry++
	return et.retry <= et.MaxRetry
}

func (et *ExponentTime) Timer() *time.Timer {
	if et.InitDuration > et.MaxDuration {
		et.InitDuration = et.MaxDuration
	}

	timer := time.NewTimer(et.InitDuration)
	et.InitDuration *= 2

	return timer
}
