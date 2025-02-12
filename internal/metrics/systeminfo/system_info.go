package systeminfo

import (
	"context"
	"encoding/json"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/sensors"
	"github.com/yusing/go-proxy/internal/metrics/period"
	"github.com/yusing/go-proxy/internal/utils/strutils"
)

type SystemInfo struct {
	Timestamp   time.Time
	CPUAverage  float64
	Memory      *mem.VirtualMemoryStat
	Disk        *disk.UsageStat
	NetworkIO   *net.IOCountersStat
	NetworkUp   float64
	NetworkDown float64
	Sensors     []sensors.TemperatureStat
}

var Poller = period.NewPoller("system_info", getSystemInfo)

func init() {
	Poller.Start()
}

func getSystemInfo(ctx context.Context, lastResult *SystemInfo) (*SystemInfo, error) {
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
	networkIO, err := net.IOCounters(false)
	if err != nil {
		return nil, err
	}
	sensors, err := sensors.SensorsTemperatures()
	if err != nil {
		return nil, err
	}
	var networkUp, networkDown float64
	if lastResult != nil {
		interval := time.Since(lastResult.Timestamp).Seconds()
		networkUp = float64(networkIO[0].BytesSent-lastResult.NetworkIO.BytesSent) / interval
		networkDown = float64(networkIO[0].BytesRecv-lastResult.NetworkIO.BytesRecv) / interval
	}

	return &SystemInfo{
		Timestamp:   time.Now(),
		CPUAverage:  cpuAverage[0],
		Memory:      memoryInfo,
		Disk:        diskInfo,
		NetworkIO:   &networkIO[0],
		NetworkUp:   networkUp,
		NetworkDown: networkDown,
		Sensors:     sensors,
	}, nil
}

func (s *SystemInfo) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"timestamp":   s.Timestamp.Unix(),
		"time":        strutils.FormatTime(s.Timestamp),
		"cpu_average": s.CPUAverage,
		"memory": map[string]any{
			"total":        s.Memory.Total,
			"available":    s.Memory.Available,
			"used":         s.Memory.Used,
			"used_percent": s.Memory.UsedPercent,
		},
		"disk": map[string]any{
			"path":         s.Disk.Path,
			"fstype":       s.Disk.Fstype,
			"total":        s.Disk.Total,
			"used":         s.Disk.Used,
			"used_percent": s.Disk.UsedPercent,
			"free":         s.Disk.Free,
		},
		"network": map[string]any{
			"name":           s.NetworkIO.Name,
			"bytes_sent":     s.NetworkIO.BytesSent,
			"bytes_recv":     s.NetworkIO.BytesRecv,
			"upload_speed":   s.NetworkUp,
			"download_speed": s.NetworkDown,
		},
		"sensors": s.Sensors,
	})
}
