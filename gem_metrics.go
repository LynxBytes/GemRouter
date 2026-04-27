package gem

import (
	"bufio"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const histogramBucketCount = 11

var histogramBuckets = [histogramBucketCount]float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10}

type histBucket struct {
	mu     sync.Mutex
	counts [histogramBucketCount]int64
	sum    float64
	total  int64
}

func (h *histBucket) observe(dur float64) {
	h.mu.Lock()
	for i := 0; i < histogramBucketCount; i++ {
		if dur <= histogramBuckets[i] {
			h.counts[i]++
		}
	}
	h.sum += dur
	h.total++
	h.mu.Unlock()
}

type gemMetrics struct {
	inFlight atomic.Int64

	cntMu    sync.RWMutex
	counters map[string]*atomic.Int64

	histMu     sync.RWMutex
	histograms map[string]*histBucket
}

func newGemMetrics() *gemMetrics {
	return &gemMetrics{
		counters:   make(map[string]*atomic.Int64),
		histograms: make(map[string]*histBucket),
	}
}

func (m *gemMetrics) record(method, pattern, status string, dur float64) {
	cKey := method + "\x00" + pattern + "\x00" + status
	m.cntMu.RLock()
	c, ok := m.counters[cKey]
	m.cntMu.RUnlock()
	if !ok {
		m.cntMu.Lock()
		if c, ok = m.counters[cKey]; !ok {
			c = &atomic.Int64{}
			m.counters[cKey] = c
		}
		m.cntMu.Unlock()
	}
	c.Add(1)

	hKey := method + "\x00" + pattern
	m.histMu.RLock()
	h, ok := m.histograms[hKey]
	m.histMu.RUnlock()
	if !ok {
		m.histMu.Lock()
		if h, ok = m.histograms[hKey]; !ok {
			h = &histBucket{}
			m.histograms[hKey] = h
		}
		m.histMu.Unlock()
	}
	h.observe(dur)
}

func (m *gemMetrics) middleware() Middleware {
	return func(next GemHandler) GemHandler {
		return func(ctx *GemContext) {
			start := time.Now()
			m.inFlight.Add(1)
			defer m.inFlight.Add(-1)

			next(ctx)

			pattern := ctx.Pattern
			if i := strings.IndexByte(pattern, ' '); i >= 0 {
				pattern = pattern[i+1:]
			}
			if pattern == "" {
				pattern = ctx.Path()
			}

			m.record(ctx.Method(), pattern, strconv.Itoa(ctx.StatusCode()), time.Since(start).Seconds())
		}
	}
}

func (m *gemMetrics) handler() GemHandler {
	return func(ctx *GemContext) {
		ctx.Writer.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		ctx.Writer.WriteHeader(http.StatusOK)

		bw := bufio.NewWriter(ctx.Writer)
		m.writeTo(bw)
		_ = bw.Flush()
	}
}

func (m *gemMetrics) writeTo(w *bufio.Writer) {
	fmt.Fprintf(w, "# HELP http_requests_in_flight Current number of HTTP requests being handled.\n")
	fmt.Fprintf(w, "# TYPE http_requests_in_flight gauge\n")
	fmt.Fprintf(w, "http_requests_in_flight %d\n\n", m.inFlight.Load())

	fmt.Fprintf(w, "# HELP http_requests_total Total number of HTTP requests processed.\n")
	fmt.Fprintf(w, "# TYPE http_requests_total counter\n")
	m.cntMu.RLock()
	for key, c := range m.counters {
		parts := strings.SplitN(key, "\x00", 3)
		fmt.Fprintf(w, "http_requests_total{method=%q,pattern=%q,status=%q} %d\n",
			parts[0], parts[1], parts[2], c.Load())
	}
	m.cntMu.RUnlock()

	fmt.Fprintf(w, "\n# HELP http_request_duration_seconds HTTP request latency in seconds.\n")
	fmt.Fprintf(w, "# TYPE http_request_duration_seconds histogram\n")
	m.histMu.RLock()
	for key, h := range m.histograms {
		parts := strings.SplitN(key, "\x00", 2)
		method, pattern := parts[0], parts[1]
		h.mu.Lock()
		for i := 0; i < histogramBucketCount; i++ {
			fmt.Fprintf(w, "http_request_duration_seconds_bucket{method=%q,pattern=%q,le=%q} %d\n",
				method, pattern, strconv.FormatFloat(histogramBuckets[i], 'f', -1, 64), h.counts[i])
		}
		fmt.Fprintf(w, "http_request_duration_seconds_bucket{method=%q,pattern=%q,le=\"+Inf\"} %d\n",
			method, pattern, h.total)
		fmt.Fprintf(w, "http_request_duration_seconds_sum{method=%q,pattern=%q} %s\n",
			method, pattern, strconv.FormatFloat(h.sum, 'f', -1, 64))
		fmt.Fprintf(w, "http_request_duration_seconds_count{method=%q,pattern=%q} %d\n",
			method, pattern, h.total)
		h.mu.Unlock()
	}
	m.histMu.RUnlock()
}
