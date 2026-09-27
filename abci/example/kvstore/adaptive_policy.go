package kvstore

import "math"

type AdaptivePolicy struct {
	MinX        uint64
	MaxX        uint64
	CPUWeight   float64
	MemoryWeight float64
	DiskWeight  float64
}

func NewAdaptivePolicy(
	minX, maxX uint64,
	cpuWeight, memoryWeight, diskWeight float64,
) *AdaptivePolicy {

	totalWeight := cpuWeight + memoryWeight + diskWeight

	// Prevent an invalid configuration.
	if totalWeight <= 0 {
		cpuWeight = 1.0 / 3.0
		memoryWeight = 1.0 / 3.0
		diskWeight = 1.0 / 3.0
	} else {
		// Normalize so the weights always add up to 1.
		cpuWeight /= totalWeight
		memoryWeight /= totalWeight
		diskWeight /= totalWeight
	}

	return &AdaptivePolicy{
		MinX:         minX,
		MaxX:         maxX,
		CPUWeight:    cpuWeight,
		MemoryWeight: memoryWeight,
		DiskWeight:   diskWeight,
	}
}

// PressureScore converts CPU, memory and disk pressure
// into a single score between 0 and 100.
func (p *AdaptivePolicy) PressureScore(t Telemetry) float64 {
	// Free disk is good, so convert it into disk pressure.
	diskPressure := 100.0 - t.DiskFreePercent

	score :=
		(p.CPUWeight * t.CPUPercent) +
			(p.MemoryWeight * t.MemoryPercent) +
			(p.DiskWeight * diskPressure)

	if score < 0 {
		return 0
	}

	if score > 100 {
		return 100
	}

	return score
}

// Calculate determines the target retention window.
//
// Higher pressure -> smaller X.
// Lower pressure -> larger target X.
//
// X is never increased during the current run because
// previously pruned blocks cannot automatically come back.
func (p *AdaptivePolicy) Calculate(currentX uint64, t Telemetry) uint64 {
	if p.MinX == 0 || p.MaxX <= p.MinX {
		return currentX
	}

	score := p.PressureScore(t)

	rangeX := float64(p.MaxX - p.MinX)

	targetX := float64(p.MaxX) -
		(score/100.0)*rangeX

	target := uint64(math.Round(targetX))

	if target < p.MinX {
		target = p.MinX
	}

	if target > p.MaxX {
		target = p.MaxX
	}

	// Never increase X during this run.
	if target > currentX {
		return currentX
	}

	return target
}