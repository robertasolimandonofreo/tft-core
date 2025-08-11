package internal

import (
	"context"
	"fmt"

	"os"
	"runtime"
	"runtime/pprof"
	"time"
)

type Profiler struct {
	enabled bool
	logger  *Logger
}

func NewProfiler(logger *Logger) *Profiler {
	enabled := os.Getenv("ENABLE_PROFILING") == "true"
	return &Profiler{
		enabled: enabled,
		logger:  logger,
	}
}

func (p *Profiler) StartMemoryProfiling() {
	if !p.enabled {
		return
	}

	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			p.captureMemoryProfile()
		}
	}()

	p.logger.Info("memory_profiling_started").
		Component("profiler").
		Operation("start_memory").
		Log()
}

func (p *Profiler) captureMemoryProfile() {
	filename := fmt.Sprintf("mem_%d.prof", time.Now().Unix())

	f, err := os.Create(filename)
	if err != nil {
		p.logger.Error("memory_profile_create_failed").
			Component("profiler").
			Operation("capture_memory").
			Err(err).
			Log()
		return
	}
	defer f.Close()

	runtime.GC()

	if err := pprof.WriteHeapProfile(f); err != nil {
		p.logger.Error("memory_profile_write_failed").
			Component("profiler").
			Operation("capture_memory").
			Err(err).
			Log()
		return
	}

	p.logger.Info("memory_profile_captured").
		Component("profiler").
		Operation("capture_memory").
		Meta("filename", filename).
		Log()
}

func (p *Profiler) StartCPUProfiling() {
	if !p.enabled {
		return
	}

	filename := fmt.Sprintf("cpu_%d.prof", time.Now().Unix())
	f, err := os.Create(filename)
	if err != nil {
		p.logger.Error("cpu_profile_create_failed").
			Component("profiler").
			Operation("start_cpu").
			Err(err).
			Log()
		return
	}

	if err := pprof.StartCPUProfile(f); err != nil {
		p.logger.Error("cpu_profile_start_failed").
			Component("profiler").
			Operation("start_cpu").
			Err(err).
			Log()
		f.Close()
		return
	}

	p.logger.Info("cpu_profiling_started").
		Component("profiler").
		Operation("start_cpu").
		Meta("filename", filename).
		Log()

	go func() {
		time.Sleep(30 * time.Second)
		pprof.StopCPUProfile()
		f.Close()

		p.logger.Info("cpu_profiling_stopped").
			Component("profiler").
			Operation("stop_cpu").
			Meta("filename", filename).
			Log()
	}()
}

func (p *Profiler) LogMemoryStats() {
	if !p.enabled {
		return
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	p.logger.Info("memory_stats").
		Component("profiler").
		Operation("log_stats").
		Meta("alloc_mb", bToMb(m.Alloc)).
		Meta("total_alloc_mb", bToMb(m.TotalAlloc)).
		Meta("sys_mb", bToMb(m.Sys)).
		Meta("gc_cycles", m.NumGC).
		Meta("goroutines", runtime.NumGoroutine()).
		Log()
}

func (p *Profiler) StartPeriodicMemoryLogging() {
	if !p.enabled {
		return
	}

	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			p.LogMemoryStats()
		}
	}()

	p.logger.Info("periodic_memory_logging_started").
		Component("profiler").
		Operation("start_periodic").
		Log()
}

func (p *Profiler) CaptureGoroutineProfile() {
	if !p.enabled {
		return
	}

	filename := fmt.Sprintf("goroutine_%d.prof", time.Now().Unix())
	f, err := os.Create(filename)
	if err != nil {
		p.logger.Error("goroutine_profile_create_failed").
			Component("profiler").
			Operation("capture_goroutine").
			Err(err).
			Log()
		return
	}
	defer f.Close()

	if err := pprof.Lookup("goroutine").WriteTo(f, 0); err != nil {
		p.logger.Error("goroutine_profile_write_failed").
			Component("profiler").
			Operation("capture_goroutine").
			Err(err).
			Log()
		return
	}

	p.logger.Info("goroutine_profile_captured").
		Component("profiler").
		Operation("capture_goroutine").
		Meta("filename", filename).
		Meta("goroutines", runtime.NumGoroutine()).
		Log()
}

func (p *Profiler) MonitorHighMemoryUsage(thresholdMB uint64) {
	if !p.enabled {
		return
	}

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			currentMB := bToMb(m.Alloc)
			if currentMB > thresholdMB {
				p.logger.Warn("high_memory_usage_detected").
					Component("profiler").
					Operation("monitor_memory").
					Meta("current_mb", currentMB).
					Meta("threshold_mb", thresholdMB).
					Meta("goroutines", runtime.NumGoroutine()).
					Log()

				p.captureMemoryProfile()
				p.CaptureGoroutineProfile()
			}
		}
	}()

	p.logger.Info("memory_monitor_started").
		Component("profiler").
		Operation("start_monitor").
		Meta("threshold_mb", thresholdMB).
		Log()
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

func (p *Profiler) ProfileFunction(ctx context.Context, name string, fn func() error) error {
	if !p.enabled {
		return fn()
	}

	start := time.Now()
	var m1, m2 runtime.MemStats

	runtime.ReadMemStats(&m1)
	err := fn()
	runtime.ReadMemStats(&m2)

	duration := time.Since(start)
	allocDiff := m2.TotalAlloc - m1.TotalAlloc

	p.logger.Info("function_profiled").
		Component("profiler").
		Operation("profile_function").
		Meta("function_name", name).
		Duration(duration).
		Meta("memory_alloc_bytes", allocDiff).
		Meta("memory_alloc_mb", allocDiff/1024/1024).
		Log()

	return err
}
