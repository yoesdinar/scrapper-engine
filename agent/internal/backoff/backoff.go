package backoff

import (
	"math/rand"
	"time"
)

type Backoff struct {
	InitialInterval time.Duration
	MaxInterval     time.Duration
	Multiplier      float64
	currentInterval time.Duration
}

func New(initial, max time.Duration, multiplier float64) *Backoff {
	return &Backoff{
		InitialInterval: initial,
		MaxInterval:     max,
		Multiplier:      multiplier,
	}
}

// Next returns the next backoff duration
func (b *Backoff) Next() time.Duration {
	if b.currentInterval == 0 {
		b.currentInterval = b.InitialInterval
	} else {
		b.currentInterval = time.Duration(float64(b.currentInterval) * b.Multiplier)
		if b.currentInterval > b.MaxInterval {
			b.currentInterval = b.MaxInterval
		}
	}

	// Add jitter: ±10%
	jitter := time.Duration(rand.Float64()*0.2*float64(b.currentInterval)) -
		time.Duration(0.1*float64(b.currentInterval))

	return b.currentInterval + jitter
}

// Reset resets the backoff to initial state
func (b *Backoff) Reset() {
	b.currentInterval = 0
}
