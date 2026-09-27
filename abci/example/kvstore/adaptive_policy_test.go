package kvstore

import "testing"

func TestAdaptivePolicy(t *testing.T) {
	policy := NewAdaptivePolicy(100, 200, 0.40, 0.20, 0.40)

	tests := []struct {
		name        string
		currentX    uint64
		telemetry   Telemetry
		minExpected uint64
		maxExpected uint64
	}{
		{
			name:     "low pressure",
			currentX: 200,
			telemetry: Telemetry{
				CPUPercent:       10,
				MemoryPercent:    20,
				DiskFreePercent: 80,
			},
			minExpected: 100,
			maxExpected: 200,
		},
		{
			name:     "high pressure",
			currentX: 200,
			telemetry: Telemetry{
				CPUPercent:       90,
				MemoryPercent:    90,
				DiskFreePercent: 10,
			},
			minExpected: 110,
			maxExpected: 110,
		},
		{
			name:     "cannot increase X",
			currentX: 120,
			telemetry: Telemetry{
				CPUPercent:       10,
				MemoryPercent:    20,
				DiskFreePercent: 90,
			},
			minExpected: 120,
			maxExpected: 120,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := policy.Calculate(tt.currentX, tt.telemetry)

			if got < tt.minExpected || got > tt.maxExpected {
				t.Fatalf(
					"expected X between %d and %d, got %d",
					tt.minExpected,
					tt.maxExpected,
					got,
				)
			}
		})
	}
}