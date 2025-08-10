package server

import "time"

type Clock interface {
	Now() time.Time
}

type realClock struct{}

func NewRealClock() Clock {
	return realClock{}
}

func (c realClock) Now() time.Time {
	return time.Now()
}

type fakeClock struct {
	t time.Time
}

func NewFakeClock(t time.Time) Clock {
	return &fakeClock{t: t}
}

func (c fakeClock) Now() time.Time {
	return c.t
}
