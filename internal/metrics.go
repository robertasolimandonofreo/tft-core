package internal

import (
	"sort"
	"sync"
	"time"
)

type MetricsCollector struct {
	logger *Logger
	
	requestCount     map[string]int64
	requestDuration  map[string][]int64
	cacheHits        int64
	cacheMisses      int64
	apiErrors        map[string]int64
	workerQueueDepth map[string]int64
	
	mu sync.RWMutex
}

func NewMetricsCollector(logger *Logger) *MetricsCollector {
	mc := &MetricsCollector{
		logger:           logger,
		requestCount:     make(map[string]int64),
		requestDuration:  make(map[string][]int64),
		apiErrors:        make(map[string]int64),
		workerQueueDepth: make(map[string]int64),
	}
	
	go mc.startMetricsReporter()
	return mc
}

func (mc *MetricsCollector) RecordRequest(endpoint string, duration time.Duration, statusCode int) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	mc.requestCount[endpoint]++
	mc.requestDuration[endpoint] = append(mc.requestDuration[endpoint], duration.Milliseconds())
	
	if statusCode >= 400 {
		mc.apiErrors[endpoint]++
	}
	
	mc.logger.Info("request_completed").
		Component("metrics").
		Operation("record_request").
		HTTP("", endpoint, statusCode).
		Duration(duration).
		Meta("endpoint", endpoint).
		Log()
}

func (mc *MetricsCollector) RecordCacheHit(key string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	mc.cacheHits++
	
	mc.logger.Debug("cache_hit").
		Component("metrics").
		Operation("record_cache").
		Cache(true, key).
		Log()
}

func (mc *MetricsCollector) RecordCacheMiss(key string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	mc.cacheMisses++
	
	mc.logger.Debug("cache_miss").
		Component("metrics").
		Operation("record_cache").
		Cache(false, key).
		Log()
}

func (mc *MetricsCollector) RecordWorkerQueueDepth(workerType string, depth int) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	mc.workerQueueDepth[workerType] = int64(depth)
	
	mc.logger.Debug("worker_queue_depth").
		Component("metrics").
		Operation("record_queue").
		Worker(workerType, "", depth).
		Log()
}

func (mc *MetricsCollector) startMetricsReporter() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		mc.reportMetrics()
	}
}

func (mc *MetricsCollector) reportMetrics() {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	totalRequests := mc.sumMapValues(mc.requestCount)
	totalErrors := mc.sumMapValues(mc.apiErrors)
	cacheHitRate := mc.calculateCacheHitRate()
	
	mc.logger.Info("metrics_report").
		Component("metrics").
		Operation("report").
		Meta("total_requests", totalRequests).
		Meta("total_errors", totalErrors).
		Meta("cache_hits", mc.cacheHits).
		Meta("cache_misses", mc.cacheMisses).
		Meta("cache_hit_rate_percent", cacheHitRate).
		Meta("worker_queue_depths", mc.workerQueueDepth).
		Log()
	
	mc.reportEndpointPerformance()
}

func (mc *MetricsCollector) reportEndpointPerformance() {
	for endpoint, durations := range mc.requestDuration {
		if len(durations) == 0 {
			continue
		}
		
		avg := mc.calculateAverage(durations)
		p95 := mc.calculatePercentile(durations, 0.95)
		
		mc.logger.Info("endpoint_performance").
			Component("metrics").
			Operation("performance_report").
			Meta("endpoint", endpoint).
			Meta("request_count", mc.requestCount[endpoint]).
			Meta("avg_duration_ms", avg).
			Meta("p95_duration_ms", p95).
			Meta("error_count", mc.apiErrors[endpoint]).
			Log()
	}
}

func (mc *MetricsCollector) sumMapValues(m map[string]int64) int64 {
	sum := int64(0)
	for _, count := range m {
		sum += count
	}
	return sum
}

func (mc *MetricsCollector) calculateCacheHitRate() float64 {
	total := mc.cacheHits + mc.cacheMisses
	if total == 0 {
		return 0
	}
	return float64(mc.cacheHits) / float64(total) * 100
}

func (mc *MetricsCollector) calculateAverage(values []int64) float64 {
	if len(values) == 0 {
		return 0
	}
	
	sum := int64(0)
	for _, v := range values {
		sum += v
	}
	
	return float64(sum) / float64(len(values))
}

func (mc *MetricsCollector) calculatePercentile(values []int64, percentile float64) int64 {
	if len(values) == 0 {
		return 0
	}
	
	sortedValues := make([]int64, len(values))
	copy(sortedValues, values)
	sort.Slice(sortedValues, func(i, j int) bool {
		return sortedValues[i] < sortedValues[j]
	})
	
	index := int(percentile * float64(len(sortedValues)-1))
	return sortedValues[index]
}

func (mc *MetricsCollector) GetMetrics() map[string]interface{} {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	return map[string]interface{}{
		"cache": map[string]interface{}{
			"hits":     mc.cacheHits,
			"misses":   mc.cacheMisses,
			"hit_rate": mc.calculateCacheHitRate(),
		},
		"requests":     mc.requestCount,
		"errors":       mc.apiErrors,
		"queue_depths": mc.workerQueueDepth,
	}
}