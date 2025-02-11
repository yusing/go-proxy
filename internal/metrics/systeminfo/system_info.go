package systeminfo

import (
	"context"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/sensors"
	"github.com/yusing/go-proxy/internal/metrics/period"
)

type SystemInfo struct {
	Timestamp  time.Time
	CPUAverage float64
	Memory     *mem.VirtualMemoryStat
	Disk       *disk.UsageStat
	Network    *net.IOCountersStat
	Sensors    []sensors.TemperatureStat
}

var Poller = period.NewPoller("system_info", 1*time.Second, getSystemInfo)

func init() {
	Poller.Start()
}

func getSystemInfo(ctx context.Context) (*SystemInfo, error) {
	memoryInfo, err := mem.VirtualMemory()
	if err != nil {
		return nil, err
	}
	cpuAverage, err := cpu.PercentWithContext(ctx, 150*time.Millisecond, false)
	if err != nil {
		return nil, err
	}
	diskInfo, err := disk.Usage("/")
	if err != nil {
		return nil, err
	}
	networkInfo, err := net.IOCounters(false)
	if err != nil {
		return nil, err
	}
	sensors, err := sensors.SensorsTemperatures()
	if err != nil {
		return nil, err
	}

	return &SystemInfo{
		Timestamp:  time.Now(),
		CPUAverage: cpuAverage[0],
		Memory:     memoryInfo,
		Disk:       diskInfo,
		Network:    &networkInfo[0],
		Sensors:    sensors,
	}, nil
}
