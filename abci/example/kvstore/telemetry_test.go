//go:build windows

package kvstore

import (
	"testing"
	"time"
)

func TestTelemetrySample(t *testing.T) {
	collector := &TelemetryCollector{}

	first, err := collector.Sample()
	if err != nil {
		t.Fatalf("first telemetry sample failed: %v", err)
	}

	// Wait so CPU utilization has an interval to calculate over.
	time.Sleep(1 * time.Second)

	second, err := collector.Sample()
	if err != nil {
		t.Fatalf("second telemetry sample failed: %v", err)
	}

	t.Logf(
		"Telemetry: CPU=%.2f%%, RAM=%.2f%%, DiskFree=%.2f%%",
		second.CPUPercent,
		second.MemoryPercent,
		second.DiskFreePercent,
	)

	if second.CPUPercent < 0 || second.CPUPercent > 100 {
		t.Fatalf("CPU percentage out of range: %.2f", second.CPUPercent)
	}

	if second.MemoryPercent < 0 || second.MemoryPercent > 100 {
		t.Fatalf("memory percentage out of range: %.2f", second.MemoryPercent)
	}

	if second.DiskFreePercent < 0 || second.DiskFreePercent > 100 {
		t.Fatalf("disk free percentage out of range: %.2f", second.DiskFreePercent)
	}

	_ = first
}