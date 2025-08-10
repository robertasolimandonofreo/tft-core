package internal

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLoggingMiddleware_Handler(t *testing.T) {
	logger := createTestLogger()
	metrics := NewMetricsCollector(logger)
	middleware := NewLoggingMiddleware(logger, metrics)
	
	handler := middleware.Handler(func(w http.ResponseWriter, r *http.Request) {
		requestID := GetRequestID(r.Context())
		if requestID == "" {
			t.Error("request ID should be set in context")
		}
		
		startTime := GetStartTime(r.Context())
		if startTime.IsZero() {
			t.Error("start time should be set in context")
		}
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "test-agent")
	w := httptest.NewRecorder()
	
	handler(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	
	body := w.Body.String()
	if body != "OK" {
		t.Errorf("expected body 'OK', got %s", body)
	}
}

func TestLoggingMiddleware_WithCustomStatusCode(t *testing.T) {
	logger := createTestLogger()
	metrics := NewMetricsCollector(logger)
	middleware := NewLoggingMiddleware(logger, metrics)
	
	handler := middleware.Handler(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
	})
	
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	
	handler(w, req)
	
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestLoggingMiddleware_MetricsRecording(t *testing.T) {
	logger := createTestLogger()
	metrics := NewMetricsCollector(logger)
	middleware := NewLoggingMiddleware(logger, metrics)
	
	handler := middleware.Handler(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})
	
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	
	start := time.Now()
	handler(w, req)
	duration := time.Since(start)
	
	metrics.mu.RLock()
	requestCount := metrics.requestCount["/test"]
	durations := metrics.requestDuration["/test"]
	metrics.mu.RUnlock()
	
	if requestCount != 1 {
		t.Errorf("expected 1 request recorded, got %d", requestCount)
	}
	
	if len(durations) != 1 {
		t.Errorf("expected 1 duration recorded, got %d", len(durations))
	}
	
	if durations[0] < 10 || durations[0] > duration.Milliseconds() {
		t.Errorf("duration %d ms seems incorrect (should be >= 10ms)", durations[0])
	}
}

func TestResponseWriter_WriteHeader(t *testing.T) {
	w := httptest.NewRecorder()
	wrapped := &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
	
	wrapped.WriteHeader(http.StatusNotFound)
	
	if wrapped.statusCode != http.StatusNotFound {
		t.Errorf("expected status code 404, got %d", wrapped.statusCode)
	}
	
	if w.Code != http.StatusNotFound {
		t.Errorf("expected underlying writer status 404, got %d", w.Code)
	}
}

func TestResponseWriter_DefaultStatusCode(t *testing.T) {
	w := httptest.NewRecorder()
	wrapped := &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
	
	wrapped.Write([]byte("test"))
	
	if wrapped.statusCode != http.StatusOK {
		t.Errorf("expected default status code 200, got %d", wrapped.statusCode)
	}
}

func TestGetRequestID_WithValue(t *testing.T) {
	ctx := context.WithValue(context.Background(), RequestIDKey, "test-id-123")
	
	requestID := GetRequestID(ctx)
	if requestID != "test-id-123" {
		t.Errorf("expected request ID 'test-id-123', got %s", requestID)
	}
}

func TestGetRequestID_WithoutValue(t *testing.T) {
	ctx := context.Background()
	
	requestID := GetRequestID(ctx)
	if requestID != "" {
		t.Errorf("expected empty request ID, got %s", requestID)
	}
}

func TestGetRequestID_WithWrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), RequestIDKey, 123)
	
	requestID := GetRequestID(ctx)
	if requestID != "" {
		t.Errorf("expected empty request ID for wrong type, got %s", requestID)
	}
}

func TestGetStartTime_WithValue(t *testing.T) {
	now := time.Now()
	ctx := context.WithValue(context.Background(), StartTimeKey, now)
	
	startTime := GetStartTime(ctx)
	if !startTime.Equal(now) {
		t.Errorf("expected start time %v, got %v", now, startTime)
	}
}

func TestGetStartTime_WithoutValue(t *testing.T) {
	ctx := context.Background()
	
	startTime := GetStartTime(ctx)
	if !startTime.IsZero() {
		t.Errorf("expected zero time, got %v", startTime)
	}
}

func TestGetStartTime_WithWrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), StartTimeKey, "not a time")
	
	startTime := GetStartTime(ctx)
	if !startTime.IsZero() {
		t.Errorf("expected zero time for wrong type, got %v", startTime)
	}
}