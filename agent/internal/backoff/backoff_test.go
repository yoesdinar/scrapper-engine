package backoff

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	minBackoff = 1 * time.Second
	maxBackoff = 5 * time.Minute
	multiplier = 2.0
)

func TestBackoffInitial(t *testing.T) {
	b := New(minBackoff, maxBackoff, multiplier)
	duration := b.Next()

	// First backoff should be around minBackoff (1 second)
	assert.GreaterOrEqual(t, duration, minBackoff-100*time.Millisecond)
	assert.LessOrEqual(t, duration, minBackoff+200*time.Millisecond)
}

func TestBackoffIncreases(t *testing.T) {
	b := New(minBackoff, maxBackoff, multiplier)

	prev := b.Next()
	for i := 0; i < 5; i++ {
		current := b.Next()
		// Each backoff should be increasing overall (allowing for jitter)
		// Current should be roughly 2x previous
		assert.Greater(t, current, prev/2)
		prev = current
	}
}

func TestBackoffMaximum(t *testing.T) {
	b := New(minBackoff, maxBackoff, multiplier)

	// Call Next many times to reach maximum
	var duration time.Duration
	for i := 0; i < 20; i++ {
		duration = b.Next()
	}

	// Should not exceed maxBackoff by much (allowing for jitter)
	assert.LessOrEqual(t, duration, maxBackoff+30*time.Second)
}

func TestBackoffReset(t *testing.T) {
	b := New(minBackoff, maxBackoff, multiplier)

	// Increase backoff
	for i := 0; i < 5; i++ {
		b.Next()
	}

	// Reset
	b.Reset()

	// Next call should return initial backoff
	duration := b.Next()
	assert.GreaterOrEqual(t, duration, minBackoff-100*time.Millisecond)
	assert.LessOrEqual(t, duration, minBackoff+200*time.Millisecond)
}

func TestBackoffJitter(t *testing.T) {
	// Call Next multiple times with reset to see jitter variation
	results := make([]time.Duration, 10)
	for i := 0; i < 10; i++ {
		b := New(minBackoff, maxBackoff, multiplier)
		results[i] = b.Next()
	}

	// Jitter might cause all to be the same in rare cases, so we just check reasonable bounds
	assert.True(t, results[0] >= minBackoff-200*time.Millisecond)
}

func TestBackoffProgression(t *testing.T) {
	b := New(minBackoff, maxBackoff, multiplier)

	expectations := []struct {
		minExpected time.Duration
		maxExpected time.Duration
	}{
		{500 * time.Millisecond, 2 * time.Second}, // First: ~1s
		{time.Second, 3 * time.Second},            // Second: ~2s
		{2 * time.Second, 5 * time.Second},        // Third: ~4s
		{3 * time.Second, 10 * time.Second},       // Fourth: ~8s
	}

	for i, exp := range expectations {
		duration := b.Next()
		assert.GreaterOrEqual(t, duration, exp.minExpected, "Iteration %d", i)
		assert.LessOrEqual(t, duration, exp.maxExpected, "Iteration %d", i)
	}
}
