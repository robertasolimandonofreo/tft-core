package internal

import (
	"testing"
	"time"
)

func TestMetricsCollector_RecordRequest(t *testing.T) {
	logger := createTestLogger()
	mc := NewMetricsCollector(logger)
	
	mc.RecordRequest("/test", 100*time.Millisecond, 200)
	mc.RecordRequest("/test", 200*time.Millisecond, 200)
	mc.RecordRequest("/test", 150*time.Millisecond, 500)
	
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	if mc.requestCount["/test"] != 3 {
		t.Errorf("expected 3 requests, got %d", mc.requestCount["/test"])
	}
	
	if len(mc.requestDuration["/test"]) != 3 {
		t.Errorf("expected 3 duration records, got %d", len(mc.requestDuration["/test"]))
	}
	
	expectedDurations := []int64{100, 200, 150}
	for i, expected := range expectedDurations {
		if mc.requestDuration["/test"][i] != expected {
			t.Errorf("expected duration %d, got %d", expected, mc.requestDuration["/test"][i])
		}
	}
	
	if mc.apiErrors["/test"] != 1 {
		t.Errorf("expected 1 error, got %d", mc.apiErrors["/test"])
	}
}

func TestMetricsCollector_RecordCache(t *testing.T) {
	logger := createTestLogger()
	mc := NewMetricsCollector(logger)
	
	mc.RecordCacheHit("key1")
	mc.RecordCacheHit("key2")
	mc.RecordCacheMiss("key3")
	
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	if mc.cacheHits != 2 {
		t.Errorf("expected 2 cache hits, got %d", mc.cacheHits)
	}
	
	if mc.cacheMisses != 1 {
		t.Errorf("expected 1 cache miss, got %d", mc.cacheMisses)
	}
}

func TestMetricsCollector_RecordWorkerQueueDepth(t *testing.T) {
	logger := createTestLogger()
	mc := NewMetricsCollector(logger)
	
	mc.RecordWorkerQueueDepth("summoner-worker", 5)
	mc.RecordWorkerQueueDepth("league-worker", 10)
	mc.RecordWorkerQueueDepth("summoner-worker", 3)
	
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	if mc.workerQueueDepth["summoner-worker"] != 3 {
		t.Errorf("expected summoner-worker depth 3, got %d", mc.workerQueueDepth["summoner-worker"])
	}
	
	if mc.workerQueueDepth["league-worker"] != 10 {
		t.Errorf("expected league-worker depth 10, got %d", mc.workerQueueDepth["league-worker"])
	}
}

func TestMetricsCollector_CalculateAverage(t *testing.T) {
	logger := createTestLogger()
	mc := NewMetricsCollector(logger)
	
	tests := []struct {
		values   []int64
		expected float64
	}{
		{[]int64{}, 0},
		{[]int64{100}, 100},
		{[]int64{100, 200}, 150},
		{[]int64{100, 200, 300}, 200},
		{[]int64{1, 2, 3, 4, 5}, 3},
	}
	
	for _, tt := range tests {
		result := mc.calculateAverage(tt.values)
		if result != tt.expected {
			t.Errorf("calculateAverage(%v): expected %f, got %f", tt.values, tt.expected, result)
		}
	}
}

func TestMetricsCollector_CalculatePercentile(t *testing.T) {
	logger := createTestLogger()
	mc := NewMetricsCollector(logger)
	
	tests := []struct {
		values     []int64
		percentile float64
		expected   int64
	}{
		{[]int64{}, 0.95, 0},
		{[]int64{100}, 0.95, 100},
		{[]int64{100, 200}, 0.95, 200},
		{[]int64{100, 200, 300, 400, 500}, 0.5, 300},
		{[]int64{100, 200, 300, 400, 500}, 0.95, 500},
		{[]int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, 0.9, 9},
	}
	
	for _, tt := range tests {
		result := mc.calculatePercentile(tt.values, tt.percentile)
		if result != tt.expected {
			t.Errorf("calculatePercentile(%v, %f): expected %d, got %d", 
				tt.values, tt.percentile, tt.expected, result)
		}
	}
}

func TestMetricsCollector_GetMetrics(t *testing.T) {
	logger := createTestLogger()
	mc := NewMetricsCollector(logger)
	
	mc.RecordCacheHit("key1")
	mc.RecordCacheHit("key2")
	mc.RecordCacheMiss("key3")
	mc.RecordRequest("/test", 100*time.Millisecond, 200)
	mc.RecordRequest("/test", 150*time.Millisecond, 500)
	mc.RecordWorkerQueueDepth("worker1", 5)
	
	metrics := mc.GetMetrics()
	
	cache, ok := metrics["cache"].(map[string]interface{})
	if !ok {
		t.Fatal("expected cache metrics to be a map")
	}
	
	if cache["hits"] != int64(2) {
		t.Errorf("expected 2 cache hits, got %v", cache["hits"])
	}
	
	if cache["misses"] != int64(1) {
		t.Errorf("expected 1 cache miss, got %v", cache["misses"])
	}
	
	expectedHitRate := float64(2) / float64(3) * 100
	if cache["hit_rate"] != expectedHitRate {
		t.Errorf("expected hit rate %f, got %v", expectedHitRate, cache["hit_rate"])
	}
	
	requests, ok := metrics["requests"].(map[string]int64)
	if !ok {
		t.Fatal("expected requests metrics to be a map")
	}
	
	if requests["/test"] != 2 {
		t.Errorf("expected 2 requests, got %d", requests["/test"])
	}
	
	errors, ok := metrics["errors"].(map[string]int64)
	if !ok {
		t.Fatal("expected errors metrics to be a map")
	}
	
	if errors["/test"] != 1 {
		t.Errorf("expected 1 error, got %d", errors["/test"])
	}
	
	queueDepths, ok := metrics["queue_depths"].(map[string]int64)
	if !ok {
		t.Fatal("expected queue_depths metrics to be a map")
	}
	
	if queueDepths["worker1"] != 5 {
		t.Errorf("expected queue depth 5, got %d", queueDepths["worker1"])
	}
}

func TestMetricsCollector_CacheHitRate_EdgeCases(t *testing.T) {
	logger := createTestLogger()
	mc := NewMetricsCollector(logger)
	
	// Test with no cache operations
	metrics := mc.GetMetrics()
	cache := metrics["cache"].(map[string]interface{})
	if cache["hit_rate"] != float64(0) {
		t.Errorf("expected 0%% hit rate with no operations, got %v", cache["hit_rate"])
	}
	
	// Test with only hits
	mc.RecordCacheHit("key1")
	mc.RecordCacheHit("key2")
	metrics = mc.GetMetrics()
	cache = metrics["cache"].(map[string]interface{})
	if cache["hit_rate"] != float64(100) {
		t.Errorf("expected 100%% hit rate with only hits, got %v", cache["hit_rate"])
	}
	
	// Test with only misses
	mc2 := NewMetricsCollector(logger)
	mc2.RecordCacheMiss("key1")
	mc2.RecordCacheMiss("key2")
	metrics2 := mc2.GetMetrics()
	cache2 := metrics2["cache"].(map[string]interface{})
	if cache2["hit_rate"] != float64(0) {
		t.Errorf("expected 0%% hit rate with only misses, got %v", cache2["hit_rate"])
	}
}