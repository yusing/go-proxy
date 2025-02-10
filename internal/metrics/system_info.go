package metrics

import (
	"context"
	"encoding/json"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/yusing/go-proxy/internal/utils/strutils"
)

type (
	SystemInfo struct {
		CPUAverage float64
		Memory     *mem.VirtualMemoryStat
		Disk       *disk.UsageStat
	}
)

func GetSystemInfo(ctx context.Context) (*SystemInfo, error) {
	memoryInfo, err := mem.VirtualMemory()
	if err != nil {
		return nil, err
	}
	cpuAverage, err := cpu.PercentWithContext(ctx, time.Second, false)
	if err != nil {
		return nil, err
	}

	diskInfo, err := disk.Usage("/")
	if err != nil {
		return nil, err
	}

	return &SystemInfo{
		CPUAverage: cpuAverage[0],
		Memory:     memoryInfo,
		Disk:       diskInfo,
	}, nil
}

func (info *SystemInfo) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"cpu_average": info.CPUAverage,
		"memory": map[string]interface{}{
			"total":        strutils.FormatByteSize(info.Memory.Total),
			"available":    strutils.FormatByteSize(info.Memory.Available),
			"used":         strutils.FormatByteSize(info.Memory.Used),
			"used_percent": info.Memory.UsedPercent,
			"free":         strutils.FormatByteSize(info.Memory.Free),
		},
		"disk": map[string]interface{}{
			"total":        strutils.FormatByteSize(info.Disk.Total),
			"used":         strutils.FormatByteSize(info.Disk.Used),
			"used_percent": info.Disk.UsedPercent,
			"free":         strutils.FormatByteSize(info.Disk.Free),
			"fs_type":      info.Disk.Fstype,
		},
	})
}
