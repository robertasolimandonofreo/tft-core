package internal

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type contextKey string

const (
	RequestIDKey contextKey = "request_id"
	StartTimeKey contextKey = "start_time"
)

type LoggingMiddleware struct {
	logger  *Logger
	metrics *MetricsCollector
}

func NewLoggingMiddleware(logger *Logger, metrics *MetricsCollector) *LoggingMiddleware {
	return &LoggingMiddleware{
		logger:  logger,
		metrics: metrics,
	}
}

func (lm *LoggingMiddleware) Handler(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		requestID := uuid.New().String()
		
		ctx := context.WithValue(r.Context(), RequestIDKey, requestID)
		ctx = context.WithValue(ctx, StartTimeKey, startTime)
		r = r.WithContext(ctx)
		
		lm.logger.Info("request_started").
			Component("http").
			Operation("handle_request").
			HTTP(r.Method, r.URL.Path, 0).
			Request(r.UserAgent(), r.RemoteAddr, requestID).
			Log()
		
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}
		
		next(wrapped, r)
		
		duration := time.Since(startTime)
		
		lm.logger.Info("request_completed").
			Component("http").
			Operation("handle_request").
			HTTP(r.Method, r.URL.Path, wrapped.statusCode).
			Request(r.UserAgent(), r.RemoteAddr, requestID).
			Duration(duration).
			Log()
		
		if lm.metrics != nil {
			lm.metrics.RecordRequest(r.URL.Path, duration, wrapped.statusCode)
		}
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return ""
}

func GetStartTime(ctx context.Context) time.Time {
	if t, ok := ctx.Value(StartTimeKey).(time.Time); ok {
		return t
	}
	return time.Time{}
}