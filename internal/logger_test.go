package internal

import (
	"bytes"
	"encoding/json"
	"log"
	"strings"
	"testing"
	"time"
)

func TestLogger_NewLogger(t *testing.T) {
	cfg := &Config{
		LogLevel: "debug",
		AppEnv:   "test",
	}

	logger := NewLogger(cfg)

	if logger.level != LogLevelDebug {
		t.Errorf("expected level debug, got %s", logger.level)
	}
	if logger.service != "tft-core" {
		t.Errorf("expected service tft-core, got %s", logger.service)
	}
	if logger.environment != "test" {
		t.Errorf("expected environment test, got %s", logger.environment)
	}
}

func TestLogger_ShouldLog(t *testing.T) {
	tests := []struct {
		loggerLevel LogLevel
		messageLevel LogLevel
		shouldLog   bool
	}{
		{LogLevelDebug, LogLevelDebug, true},
		{LogLevelDebug, LogLevelInfo, true},
		{LogLevelDebug, LogLevelWarn, true},
		{LogLevelDebug, LogLevelError, true},
		{LogLevelInfo, LogLevelDebug, false},
		{LogLevelInfo, LogLevelInfo, true},
		{LogLevelInfo, LogLevelWarn, true},
		{LogLevelInfo, LogLevelError, true},
		{LogLevelWarn, LogLevelDebug, false},
		{LogLevelWarn, LogLevelInfo, false},
		{LogLevelWarn, LogLevelWarn, true},
		{LogLevelWarn, LogLevelError, true},
		{LogLevelError, LogLevelDebug, false},
		{LogLevelError, LogLevelInfo, false},
		{LogLevelError, LogLevelWarn, false},
		{LogLevelError, LogLevelError, true},
	}

	for _, tt := range tests {
		logger := &Logger{level: tt.loggerLevel}
		result := logger.shouldLog(tt.messageLevel)
		if result != tt.shouldLog {
			t.Errorf("level %s should log %s: expected %v, got %v", 
				tt.loggerLevel, tt.messageLevel, tt.shouldLog, result)
		}
	}
}

func TestLogger_LogOutput(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		level:       LogLevelInfo,
		service:     "tft-core",
		environment: "test",
		logger:      log.New(&buf, "", 0),
	}

	logger.Info("test message").
		Component("test").
		Operation("test_op").
		Duration(100 * time.Millisecond).
		Log()

	output := buf.String()
	
	if !strings.Contains(output, "test message") {
		t.Error("output should contain message")
	}
	if !strings.Contains(output, "info") {
		t.Error("output should contain level")
	}
	if !strings.Contains(output, "tft-core") {
		t.Error("output should contain service")
	}

	var logEntry LogEntry
	if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
		t.Errorf("output should be valid JSON: %v", err)
	}

	if logEntry.Message != "test message" {
		t.Errorf("expected message 'test message', got %s", logEntry.Message)
	}
	if logEntry.Component != "test" {
		t.Errorf("expected component 'test', got %s", logEntry.Component)
	}
	if logEntry.Operation != "test_op" {
		t.Errorf("expected operation 'test_op', got %s", logEntry.Operation)
	}
	if logEntry.Duration != 100 {
		t.Errorf("expected duration 100, got %d", logEntry.Duration)
	}
}

func TestLogBuilder_HTTP(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		level:       LogLevelInfo,
		service:     "tft-core",
		environment: "test",
		logger:      log.New(&buf, "", 0),
	}

	logger.Info("http request").
		HTTP("GET", "/test", 200).
		Log()

	var logEntry LogEntry
	json.Unmarshal([]byte(buf.String()), &logEntry)

	if logEntry.Method != "GET" {
		t.Errorf("expected method GET, got %s", logEntry.Method)
	}
	if logEntry.Path != "/test" {
		t.Errorf("expected path /test, got %s", logEntry.Path)
	}
	if logEntry.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", logEntry.StatusCode)
	}
}

func TestLogBuilder_Cache(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		level:       LogLevelInfo,
		service:     "tft-core",
		environment: "test",
		logger:      log.New(&buf, "", 0),
	}

	logger.Info("cache hit").
		Cache(true, "test:key").
		Log()

	var logEntry LogEntry
	json.Unmarshal([]byte(buf.String()), &logEntry)

	if logEntry.CacheHit == nil || !*logEntry.CacheHit {
		t.Error("expected cache hit to be true")
	}
	if logEntry.CacheKey != "test:key" {
		t.Errorf("expected cache key 'test:key', got %s", logEntry.CacheKey)
	}
}

func TestLogBuilder_Game(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		level:       LogLevelInfo,
		service:     "tft-core",
		environment: "test",
		logger:      log.New(&buf, "", 0),
	}

	longPUUID := "abcdefghijklmnopqrstuvwxyz1234567890"
	logger.Info("game data").
		Game(longPUUID, "BR1", "CHALLENGER").
		Log()

	var logEntry LogEntry
	json.Unmarshal([]byte(buf.String()), &logEntry)

	if !strings.HasSuffix(logEntry.PUUID, "...") {
		t.Error("long PUUID should be truncated")
	}
	if logEntry.Region != "BR1" {
		t.Errorf("expected region BR1, got %s", logEntry.Region)
	}
	if logEntry.Tier != "CHALLENGER" {
		t.Errorf("expected tier CHALLENGER, got %s", logEntry.Tier)
	}
}

func TestLogBuilder_Error(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		level:       LogLevelError,
		service:     "tft-core",
		environment: "test",
		logger:      log.New(&buf, "", 0),
	}

	testErr := NewAPIError("test error", 500)
	logger.Error("error occurred").
		Err(testErr).
		ErrorCode("API_ERROR").
		Log()

	var logEntry LogEntry
	json.Unmarshal([]byte(buf.String()), &logEntry)

	if logEntry.Error != "test error" {
		t.Errorf("expected error 'test error', got %s", logEntry.Error)
	}
	if logEntry.ErrorCode != "API_ERROR" {
		t.Errorf("expected error code 'API_ERROR', got %s", logEntry.ErrorCode)
	}
}

func TestLogBuilder_Meta(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		level:       LogLevelInfo,
		service:     "tft-core",
		environment: "test",
		logger:      log.New(&buf, "", 0),
	}

	logger.Info("with metadata").
		Meta("key1", "value1").
		Meta("key2", 42).
		Log()

	var logEntry LogEntry
	json.Unmarshal([]byte(buf.String()), &logEntry)

	if logEntry.Metadata["key1"] != "value1" {
		t.Errorf("expected metadata key1 'value1', got %v", logEntry.Metadata["key1"])
	}
	if logEntry.Metadata["key2"] != float64(42) {
		t.Errorf("expected metadata key2 42, got %v", logEntry.Metadata["key2"])
	}
}