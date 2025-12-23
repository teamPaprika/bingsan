package pool

import (
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// poolGets tracks total Get() calls per pool.
	poolGets = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "bingsan",
		Subsystem: "pool",
		Name:      "gets_total",
		Help:      "Total number of pool Get() calls",
	}, []string{"pool"})

	// poolReturns tracks successful Put() calls per pool.
	poolReturns = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "bingsan",
		Subsystem: "pool",
		Name:      "returns_total",
		Help:      "Total number of pool Put() calls (successful returns)",
	}, []string{"pool"})

	// poolDiscards tracks discarded items (too large or wrong size).
	poolDiscards = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "bingsan",
		Subsystem: "pool",
		Name:      "discards_total",
		Help:      "Total number of discarded pool items (oversized or invalid)",
	}, []string{"pool"})

	// poolMisses tracks when pool was empty and new allocation was needed.
	// Note: sync.Pool doesn't expose hit/miss directly, this is for custom pools.
	poolMisses = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "bingsan",
		Subsystem: "pool",
		Name:      "misses_total",
		Help:      "Total number of pool misses (new allocations)",
	}, []string{"pool"})
)

// PoolMetrics tracks pool utilization for observability.
type PoolMetrics struct {
	gets     atomic.Uint64
	returns  atomic.Uint64
	discards atomic.Uint64
	misses   atomic.Uint64
}

// NewPoolMetrics creates a new PoolMetrics instance.
func NewPoolMetrics() *PoolMetrics {
	return &PoolMetrics{}
}

// RecordGet increments the get counter for a pool.
func (m *PoolMetrics) RecordGet(pool string) {
	m.gets.Add(1)
	poolGets.WithLabelValues(pool).Inc()
}

// RecordReturn increments the return counter for a pool.
func (m *PoolMetrics) RecordReturn(pool string) {
	m.returns.Add(1)
	poolReturns.WithLabelValues(pool).Inc()
}

// RecordDiscard increments the discard counter for a pool.
func (m *PoolMetrics) RecordDiscard(pool string) {
	m.discards.Add(1)
	poolDiscards.WithLabelValues(pool).Inc()
}

// RecordMiss increments the miss counter for a pool.
func (m *PoolMetrics) RecordMiss(pool string) {
	m.misses.Add(1)
	poolMisses.WithLabelValues(pool).Inc()
}

// Stats returns current pool statistics.
func (m *PoolMetrics) Stats() PoolStats {
	return PoolStats{
		Gets:     m.gets.Load(),
		Returns:  m.returns.Load(),
		Discards: m.discards.Load(),
		Misses:   m.misses.Load(),
	}
}

// PoolStats contains pool utilization statistics.
type PoolStats struct {
	Gets     uint64
	Returns  uint64
	Discards uint64
	Misses   uint64
}

// HitRate calculates the estimated hit rate.
// Note: This is an approximation since sync.Pool doesn't track hits directly.
// Hit rate = (gets - misses) / gets
func (s PoolStats) HitRate() float64 {
	if s.Gets == 0 {
		return 0
	}
	hits := s.Gets - s.Misses
	return float64(hits) / float64(s.Gets)
}
