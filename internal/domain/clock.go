package domain

import "time"

// Clock provides current time; useful for deterministic tests.
type Clock interface {
	Now() time.Time
}

type RealClock struct{}

func (RealClock) Now() time.Time { return time.Now() }
