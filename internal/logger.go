package internal

import (
	"encoding/json"
	"log"
	"os"
	"time"
)

type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

type LogEntry struct {
	Timestamp  time.Time              `json:"timestamp"`
	Level      LogLevel               `json:"level"`
	Message    string                 `json:"message"`
	Service    string                 `json:"service"`
	Component  string                 `json:"component"`
	Operation  string                 `json:"operation,omitempty"`
	Duration   int64                  `json:"duration_ms,omitempty"`
	StatusCode int                    `json:"status_code,omitempty"`
	Method     string                 `json:"method,omitempty"`
	Path       string                 `json:"path,omitempty"`
	UserAgent  string                 `json:"user_agent,omitempty"`
	RemoteAddr string                 `json:"remote_addr,omitempty"`
	RequestID  string                 `json:"request_id,omitempty"`
	CacheHit   *bool                  `json:"cache_hit,omitempty"`
	CacheKey   string                 `json:"cache_key,omitempty"`
	QueueDepth int                    `json:"queue_depth,omitempty"`
	WorkerID   string                 `json:"worker_id,omitempty"`
	TaskType   string                 `json:"task_type,omitempty"`
	PUUID      string                 `json:"puuid,omitempty"`
	Region     string                 `json:"region,omitempty"`
	Tier       string                 `json:"tier,omitempty"`
	Error      string                 `json:"error,omitempty"`
	ErrorCode  string                 `json:"error_code,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

type Logger struct {
	level       LogLevel
	service     string
	environment string
	logger      *log.Logger
}

func NewLogger(cfg *Config) *Logger {
	level := LogLevel(cfg.LogLevel)
	if level == "" {
		level = LogLevelInfo
	}

	return &Logger{
		level:       level,
		service:     "tft-core",
		environment: cfg.AppEnv,
		logger:      log.New(os.Stdout, "", 0),
	}
}

func (l *Logger) shouldLog(level LogLevel) bool {
	levels := map[LogLevel]int{
		LogLevelDebug: 0,
		LogLevelInfo:  1,
		LogLevelWarn:  2,
		LogLevelError: 3,
	}
	return levels[level] >= levels[l.level]
}

func (l *Logger) log(entry LogEntry) {
	if !l.shouldLog(entry.Level) {
		return
	}

	entry.Timestamp = time.Now().UTC()
	entry.Service = l.service

	if entry.Metadata == nil {
		entry.Metadata = make(map[string]interface{})
	}
	entry.Metadata["environment"] = l.environment

	jsonData, err := json.Marshal(entry)
	if err != nil {
		log.Printf("Failed to marshal log entry: %v", err)
		return
	}

	l.logger.Println(string(jsonData))
}

func (l *Logger) Debug(message string) *LogBuilder {
	return &LogBuilder{logger: l, entry: LogEntry{Level: LogLevelDebug, Message: message}}
}

func (l *Logger) Info(message string) *LogBuilder {
	return &LogBuilder{logger: l, entry: LogEntry{Level: LogLevelInfo, Message: message}}
}

func (l *Logger) Warn(message string) *LogBuilder {
	return &LogBuilder{logger: l, entry: LogEntry{Level: LogLevelWarn, Message: message}}
}

func (l *Logger) Error(message string) *LogBuilder {
	return &LogBuilder{logger: l, entry: LogEntry{Level: LogLevelError, Message: message}}
}

type LogBuilder struct {
	logger *Logger
	entry  LogEntry
}

func (b *LogBuilder) Component(component string) *LogBuilder {
	b.entry.Component = component
	return b
}

func (b *LogBuilder) Operation(operation string) *LogBuilder {
	b.entry.Operation = operation
	return b
}

func (b *LogBuilder) Duration(duration time.Duration) *LogBuilder {
	b.entry.Duration = duration.Milliseconds()
	return b
}

func (b *LogBuilder) HTTP(method, path string, statusCode int) *LogBuilder {
	b.entry.Method = method
	b.entry.Path = path
	b.entry.StatusCode = statusCode
	return b
}

func (b *LogBuilder) Request(userAgent, remoteAddr, requestID string) *LogBuilder {
	b.entry.UserAgent = userAgent
	b.entry.RemoteAddr = remoteAddr
	b.entry.RequestID = requestID
	return b
}

func (b *LogBuilder) Cache(hit bool, key string) *LogBuilder {
	b.entry.CacheHit = &hit
	b.entry.CacheKey = key
	return b
}

func (b *LogBuilder) Worker(workerID, taskType string, queueDepth int) *LogBuilder {
	b.entry.WorkerID = workerID
	b.entry.TaskType = taskType
	b.entry.QueueDepth = queueDepth
	return b
}

func (b *LogBuilder) Game(puuid, region, tier string) *LogBuilder {
	if puuid != "" && len(puuid) > 20 {
		b.entry.PUUID = puuid[:20] + "..."
	} else {
		b.entry.PUUID = puuid
	}
	b.entry.Region = region
	b.entry.Tier = tier
	return b
}

func (b *LogBuilder) Err(err error) *LogBuilder {
	if err != nil {
		b.entry.Error = err.Error()
	}
	return b
}

func (b *LogBuilder) ErrorCode(code string) *LogBuilder {
	b.entry.ErrorCode = code
	return b
}

func (b *LogBuilder) Meta(key string, value interface{}) *LogBuilder {
	if b.entry.Metadata == nil {
		b.entry.Metadata = make(map[string]interface{})
	}
	b.entry.Metadata[key] = value
	return b
}

func (b *LogBuilder) Log() {
	b.logger.log(b.entry)
}
