package systeminfo

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/sensors"
	"github.com/yusing/go-proxy/internal/common"
	"github.com/yusing/go-proxy/internal/gperr"
	"github.com/yusing/go-proxy/internal/logging"
	"github.com/yusing/go-proxy/internal/metrics/period"
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

func _() { // check if this behavior is not changed
	var _ sensors.Warnings = disk.Warnings{}
}

func getSystemInfo(ctx context.Context, lastResult *SystemInfo) (*SystemInfo, error) {
	errs := gperr.NewBuilder("failed to get system info")
	var systemInfo SystemInfo
	systemInfo.Timestamp = time.Now()

	if !common.MetricsDisableCPU {
		cpuAverage, err := cpu.PercentWithContext(ctx, 150*time.Millisecond, false)
		if err != nil {
			errs.Add(err)
		} else {
			systemInfo.CPUAverage = cpuAverage[0]
		}
	}

	if !common.MetricsDisableMemory {
		memoryInfo, err := mem.VirtualMemory()
		if err != nil {
			errs.Add(err)
		}
		systemInfo.Memory = memoryInfo
	}

	if !common.MetricsDisableDisk {
		diskInfo, err := disk.Usage("/")
		if err != nil {
			errs.Add(err)
		}
		systemInfo.Disk = diskInfo
	}

	if !common.MetricsDisableNetwork {
		networkIO, err := net.IOCounters(false)
		if err != nil {
			errs.Add(err)
		} else {
			networkIO := networkIO[0]
			systemInfo.NetworkIO = &networkIO
			var networkUp, networkDown float64
			if lastResult != nil {
				interval := time.Since(lastResult.Timestamp).Seconds()
				networkUp = float64(networkIO.BytesSent-lastResult.NetworkIO.BytesSent) / interval
				networkDown = float64(networkIO.BytesRecv-lastResult.NetworkIO.BytesRecv) / interval
			}
			systemInfo.NetworkUp = networkUp
			systemInfo.NetworkDown = networkDown
		}
	}

	if !common.MetricsDisableSensors {
		sensorsInfo, err := sensors.SensorsTemperatures()
		if err != nil {
			errs.Add(err)
		}
		systemInfo.Sensors = sensorsInfo
	}

	if errs.HasError() {
		allWarnings := gperr.NewBuilder("")
		allErrors := gperr.NewBuilder("failed to get system info")
		errs.ForEach(func(err error) {
			// disk.Warnings has the same type
			// all Warnings are alias of common.Warnings from "github.com/shirou/gopsutil/v4/internal/common"
			// see line 37
			warnings := new(sensors.Warnings)
			if errors.As(err, &warnings) {
				for _, warning := range warnings.List {
					allWarnings.Add(warning)
				}
			} else {
				allErrors.Add(err)
			}
		})
		if allWarnings.HasError() {
			logging.Warn().Msg(allWarnings.String())
		}
		if allErrors.HasError() {
			return nil, allErrors.Error()
		}
	}

	return &systemInfo, nil
}

func (s *SystemInfo) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"timestamp":   s.Timestamp.Unix(),
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
