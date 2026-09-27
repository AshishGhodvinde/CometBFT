//go:build windows

package kvstore

import (
	"fmt"
	"os"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

type Telemetry struct {
	CPUPercent      float64
	MemoryPercent   float64
	DiskFreePercent float64
}

type TelemetryCollector struct {
	lastIdleTimeNS   uint64
	lastKernelTimeNS uint64
	lastUserTimeNS   uint64
	lastSample       time.Time
}

type memoryStatusEx struct {
	DwLength                uint32
	DwMemoryLoad            uint32
	UllTotalPhys             uint64
	UllAvailPhys             uint64
	UllTotalPageFile         uint64
	UllAvailPageFile         uint64
	UllTotalVirtual          uint64
	UllAvailVirtual          uint64
	UllAvailExtendedVirtual  uint64
}

var (
	kernel32              = windows.NewLazySystemDLL("kernel32.dll")
	globalMemoryStatusEx  = kernel32.NewProc("GlobalMemoryStatusEx")
	getSystemTimes       = kernel32.NewProc("GetSystemTimes")
)

func (c *TelemetryCollector) Sample() (Telemetry, error) {
	cpuPercent, err := c.sampleCPU()
	if err != nil {
		return Telemetry{}, err
	}

	memoryPercent, err := sampleMemory()
	if err != nil {
		return Telemetry{}, err
	}

	diskPercent, err := sampleDisk()
	if err != nil {
		return Telemetry{}, err
	}

	return Telemetry{
		CPUPercent:       cpuPercent,
		MemoryPercent:    memoryPercent,
		DiskFreePercent: diskPercent,
	}, nil
}

func (c *TelemetryCollector) sampleCPU() (float64, error) {
	var idleTime windows.Filetime
	var kernelTime windows.Filetime
	var userTime windows.Filetime

	ret, _, err := getSystemTimes.Call(
		uintptr(unsafe.Pointer(&idleTime)),
		uintptr(unsafe.Pointer(&kernelTime)),
		uintptr(unsafe.Pointer(&userTime)),
	)

	if ret == 0 {
		return 0, fmt.Errorf("failed to get system CPU times: %w", err)
	}

	currentIdle := uint64(idleTime.Nanoseconds())
	currentKernel := uint64(kernelTime.Nanoseconds())
	currentUser := uint64(userTime.Nanoseconds())

	now := time.Now()

	// First sample: we need another sample later
	// to calculate CPU utilization.
	if c.lastSample.IsZero() {
		c.lastIdleTimeNS = currentIdle
		c.lastKernelTimeNS = currentKernel
		c.lastUserTimeNS = currentUser
		c.lastSample = now
		return 0, nil
	}

	idleDelta := currentIdle - c.lastIdleTimeNS
	kernelDelta := currentKernel - c.lastKernelTimeNS
	userDelta := currentUser - c.lastUserTimeNS

	c.lastIdleTimeNS = currentIdle
	c.lastKernelTimeNS = currentKernel
	c.lastUserTimeNS = currentUser
	c.lastSample = now

	totalDelta := kernelDelta + userDelta

	if totalDelta == 0 {
		return 0, nil
	}

	// Windows kernel time includes idle time.
	// Therefore busy time is:
	// kernel + user - idle.
	busyDelta := kernelDelta + userDelta - idleDelta

	cpuPercent := (float64(busyDelta) / float64(totalDelta)) * 100.0

	if cpuPercent < 0 {
		cpuPercent = 0
	}

	if cpuPercent > 100 {
		cpuPercent = 100
	}

	return cpuPercent, nil
}

func sampleMemory() (float64, error) {
	var status memoryStatusEx
	status.DwLength = uint32(unsafe.Sizeof(status))

	ret, _, err := globalMemoryStatusEx.Call(
		uintptr(unsafe.Pointer(&status)),
	)

	if ret == 0 {
		return 0, fmt.Errorf("failed to get memory status: %w", err)
	}

	return float64(status.DwMemoryLoad), nil
}

func sampleDisk() (float64, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return 0, fmt.Errorf("failed to get working directory: %w", err)
	}

	path, err := windows.UTF16PtrFromString(workingDir)
	if err != nil {
		return 0, fmt.Errorf("failed to convert working directory: %w", err)
	}

	var freeBytes uint64
	var totalBytes uint64
	var totalFreeBytes uint64

	if err := windows.GetDiskFreeSpaceEx(
		path,
		&freeBytes,
		&totalBytes,
		&totalFreeBytes,
	); err != nil {
		return 0, fmt.Errorf("failed to get disk information: %w", err)
	}

	if totalBytes == 0 {
		return 0, fmt.Errorf("disk total size is zero")
	}

	freePercent := float64(freeBytes) / float64(totalBytes) * 100

	return freePercent, nil
}