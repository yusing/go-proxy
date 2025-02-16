package systeminfo

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
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

// json tags are left for tests

type (
	MemoryUsage struct {
		Total       uint64  `json:"total"`
		Available   uint64  `json:"available"`
		Used        uint64  `json:"used"`
		UsedPercent float64 `json:"used_percent"`
	}
	Disk struct {
		Path        string  `json:"path"`
		Fstype      string  `json:"fstype"`
		Total       uint64  `json:"total"`
		Free        uint64  `json:"free"`
		Used        uint64  `json:"used"`
		UsedPercent float64 `json:"used_percent"`
	}
	DiskIO struct {
		ReadBytes  uint64  `json:"read_bytes"`
		WriteBytes uint64  `json:"write_bytes"`
		ReadCount  uint64  `json:"read_count"`
		WriteCount uint64  `json:"write_count"`
		ReadSpeed  float64 `json:"read_speed"`
		WriteSpeed float64 `json:"write_speed"`
		Iops       uint64  `json:"iops"`
	}
	Network struct {
		BytesSent     uint64  `json:"bytes_sent"`
		BytesRecv     uint64  `json:"bytes_recv"`
		UploadSpeed   float64 `json:"upload_speed"`
		DownloadSpeed float64 `json:"download_speed"`
	}
	Sensor struct {
		Temperature float32 `json:"temperature"`
		High        float32 `json:"high"`
		Critical    float32 `json:"critical"`
	}
)

type SystemInfo struct {
	Timestamp  int64              `json:"timestamp"`
	CPUAverage *float64           `json:"cpu_average"`
	Memory     *MemoryUsage       `json:"memory"`
	Disks      map[string]*Disk   `json:"disks"`    // disk usage by partition
	DisksIO    map[string]*DiskIO `json:"disks_io"` // disk IO by device
	Network    *Network           `json:"network"`
	Sensors    map[string]Sensor  `json:"sensors"` // sensor temperature by key
}

const (
	queryCPUAverage         = "cpu_average"
	queryMemoryUsage        = "memory_usage"
	queryMemoryUsagePercent = "memory_usage_percent"
	queryDisksReadSpeed     = "disks_read_speed"
	queryDisksWriteSpeed    = "disks_write_speed"
	queryDisksIOPS          = "disks_iops"
	queryDiskUsage          = "disk_usage"
	queryNetworkSpeed       = "network_speed"
	queryNetworkTransfer    = "network_transfer"
	querySensorTemperature  = "sensor_temperature"
)

var Poller = period.NewPollerWithAggregator("system_info", getSystemInfo, aggregate)

func init() {
	Poller.Start()
}

func _() { // check if this behavior is not changed
	var _ sensors.Warnings = disk.Warnings{}
}

func getSystemInfo(ctx context.Context, lastResult *SystemInfo) (*SystemInfo, error) {
	errs := gperr.NewBuilder("failed to get system info")
	var s SystemInfo
	s.Timestamp = time.Now().Unix()

	if !common.MetricsDisableCPU {
		errs.Add(s.collectCPUInfo(ctx))
	}
	if !common.MetricsDisableMemory {
		errs.Add(s.collectMemoryInfo(ctx))
	}
	if !common.MetricsDisableDisk {
		errs.Add(s.collectDisksInfo(ctx, lastResult))
	}
	if !common.MetricsDisableNetwork {
		errs.Add(s.collectNetworkInfo(ctx, lastResult))
	}
	if !common.MetricsDisableSensors {
		errs.Add(s.collectSensorsInfo(ctx))
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

	return &s, nil
}

func (s *SystemInfo) collectCPUInfo(ctx context.Context) error {
	cpuAverage, err := cpu.PercentWithContext(ctx, 500*time.Millisecond, false)
	if err != nil {
		return err
	}
	s.CPUAverage = new(float64)
	*s.CPUAverage = cpuAverage[0]
	return nil
}

func (s *SystemInfo) collectMemoryInfo(ctx context.Context) error {
	memoryInfo, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return err
	}
	s.Memory = &MemoryUsage{
		Total:       memoryInfo.Total,
		Available:   memoryInfo.Available,
		Used:        memoryInfo.Used,
		UsedPercent: memoryInfo.UsedPercent,
	}
	return nil
}

func (s *SystemInfo) collectDisksInfo(ctx context.Context, lastResult *SystemInfo) error {
	ioCounters, err := disk.IOCountersWithContext(ctx)
	if err != nil {
		return err
	}
	s.DisksIO = make(map[string]*DiskIO, len(ioCounters))
	for name, io := range ioCounters {
		s.DisksIO[name] = &DiskIO{
			ReadBytes:  io.ReadBytes,
			WriteBytes: io.WriteBytes,
			ReadCount:  io.ReadCount,
			WriteCount: io.WriteCount,
		}
	}
	if lastResult != nil {
		interval := float64(time.Now().Unix() - lastResult.Timestamp)
		for name, disk := range s.DisksIO {
			if lastUsage, ok := lastResult.DisksIO[name]; ok {
				disk.ReadSpeed = float64(disk.ReadBytes-lastUsage.ReadBytes) / interval
				disk.WriteSpeed = float64(disk.WriteBytes-lastUsage.WriteBytes) / interval
				disk.Iops = (disk.ReadCount + disk.WriteCount - lastUsage.ReadCount - lastUsage.WriteCount) / uint64(interval)
			}
		}
	}

	partitions, err := disk.Partitions(false)
	if err != nil {
		return err
	}
	s.Disks = make(map[string]*Disk, len(partitions))
	errs := gperr.NewBuilder("failed to get disks info")
	for _, partition := range partitions {
		d := &Disk{
			Path:   partition.Mountpoint,
			Fstype: partition.Fstype,
		}
		diskInfo, err := disk.UsageWithContext(ctx, partition.Mountpoint)
		if err != nil {
			errs.Add(err)
			continue
		}
		d.Total = diskInfo.Total
		d.Free = diskInfo.Free
		d.Used = diskInfo.Used
		d.UsedPercent = diskInfo.UsedPercent
		s.Disks[partition.Device] = d
	}

	if errs.HasError() {
		if len(s.Disks) == 0 {
			return errs.Error()
		}
		logging.Warn().Msg(errs.String())
	}
	return nil
}

func (s *SystemInfo) collectNetworkInfo(ctx context.Context, lastResult *SystemInfo) error {
	networkIO, err := net.IOCountersWithContext(ctx, false)
	if err != nil {
		return err
	}
	s.Network = &Network{
		BytesSent: networkIO[0].BytesSent,
		BytesRecv: networkIO[0].BytesRecv,
	}
	if lastResult != nil {
		interval := float64(time.Now().Unix() - lastResult.Timestamp)
		s.Network.UploadSpeed = float64(networkIO[0].BytesSent-lastResult.Network.BytesSent) / interval
		s.Network.DownloadSpeed = float64(networkIO[0].BytesRecv-lastResult.Network.BytesRecv) / interval
	}
	return nil
}

func (s *SystemInfo) collectSensorsInfo(ctx context.Context) error {
	sensorsInfo, err := sensors.SensorsTemperatures()
	if err != nil {
		return err
	}
	s.Sensors = make(map[string]Sensor, len(sensorsInfo))
	for _, sensor := range sensorsInfo {
		s.Sensors[sensor.SensorKey] = Sensor{
			Temperature: float32(sensor.Temperature),
			High:        float32(sensor.High),
			Critical:    float32(sensor.Critical),
		}
	}
	return nil
}

// explicitly implement MarshalJSON to avoid reflection
func (s *SystemInfo) MarshalJSON() ([]byte, error) {
	var b strings.Builder
	b.Grow(1024)
	b.WriteRune('{')

	// timestamp
	b.WriteString(`"timestamp":`)
	b.WriteString(strconv.FormatInt(s.Timestamp, 10))

	// cpu_average
	b.WriteString(`,"cpu_average":`)
	if s.CPUAverage != nil {
		b.WriteString(strconv.FormatFloat(*s.CPUAverage, 'f', 2, 64))
	} else {
		b.WriteString("null")
	}

	// memory
	b.WriteString(`,"memory":`)
	if s.Memory != nil {
		b.WriteString(fmt.Sprintf(
			`{"total":%d,"available":%d,"used":%d,"used_percent":%s}`,
			s.Memory.Total,
			s.Memory.Available,
			s.Memory.Used,
			strconv.FormatFloat(s.Memory.UsedPercent, 'f', 2, 64),
		))
	} else {
		b.WriteString("null")
	}

	// disk
	b.WriteString(`,"disks":`)
	if s.Disks != nil {
		b.WriteString("{")
		first := true
		for device, disk := range s.Disks {
			if !first {
				b.WriteRune(',')
			}
			b.WriteString(fmt.Sprintf(
				`"%s":{"device":%q,"path":%q,"fstype":%q,"total":%d,"free":%d,"used":%d,"used_percent":%s}`,
				device,
				device,
				disk.Path,
				disk.Fstype,
				disk.Total,
				disk.Free,
				disk.Used,
				strconv.FormatFloat(float64(disk.UsedPercent), 'f', 2, 32),
			))
			first = false
		}
		b.WriteRune('}')
	} else {
		b.WriteString("null")
	}

	// disks_io
	b.WriteString(`,"disks_io":`)
	if s.DisksIO != nil {
		b.WriteString("{")
		first := true
		for name, usage := range s.DisksIO {
			if !first {
				b.WriteRune(',')
			}
			b.WriteString(fmt.Sprintf(
				`"%s":{"name":%q,"read_bytes":%d,"write_bytes":%d,"read_speed":%s,"write_speed":%s,"iops":%d}`,
				name,
				name,
				usage.ReadBytes,
				usage.WriteBytes,
				strconv.FormatFloat(usage.ReadSpeed, 'f', 2, 64),
				strconv.FormatFloat(usage.WriteSpeed, 'f', 2, 64),
				usage.Iops,
			))
			first = false
		}
		b.WriteRune('}')
	} else {
		b.WriteString("null")
	}

	// network
	b.WriteString(`,"network":`)
	if s.Network != nil {
		b.WriteString(fmt.Sprintf(
			`{"bytes_sent":%d,"bytes_recv":%d,"upload_speed":%s,"download_speed":%s}`,
			s.Network.BytesSent,
			s.Network.BytesRecv,
			strconv.FormatFloat(s.Network.UploadSpeed, 'f', 2, 64),
			strconv.FormatFloat(s.Network.DownloadSpeed, 'f', 2, 64),
		))
	} else {
		b.WriteString("null")
	}

	// sensors
	b.WriteString(`,"sensors":`)
	if s.Sensors != nil {
		b.WriteString("{")
		first := true
		for key, sensor := range s.Sensors {
			if !first {
				b.WriteRune(',')
			}
			b.WriteString(fmt.Sprintf(
				`%q:{"name":%q,"temperature":%s,"high":%s,"critical":%s}`,
				key,
				key,
				strconv.FormatFloat(float64(sensor.Temperature), 'f', 2, 32),
				strconv.FormatFloat(float64(sensor.High), 'f', 2, 32),
				strconv.FormatFloat(float64(sensor.Critical), 'f', 2, 32),
			))
			first = false
		}
		b.WriteRune('}')
	} else {
		b.WriteString("null")
	}

	b.WriteRune('}')
	return []byte(b.String()), nil
}

// recharts friendly
func aggregate(entries []*SystemInfo, query url.Values) (total int, result []map[string]any) {
	n := len(entries)
	switch query.Get("aggregate") {
	case queryCPUAverage:
		aggregated := make([]map[string]any, 0, n)
		for _, entry := range entries {
			if entry.CPUAverage != nil {
				aggregated = append(aggregated, map[string]any{
					"timestamp":   entry.Timestamp,
					"cpu_average": *entry.CPUAverage,
				})
			}
		}
		return len(aggregated), aggregated
	case queryMemoryUsage:
		aggregated := make([]map[string]any, 0, n)
		for _, entry := range entries {
			if entry.Memory != nil {
				aggregated = append(aggregated, map[string]any{
					"timestamp":    entry.Timestamp,
					"memory_usage": entry.Memory.Used,
				})
			}
		}
		return len(aggregated), aggregated
	case queryMemoryUsagePercent:
		aggregated := make([]map[string]any, 0, n)
		for _, entry := range entries {
			if entry.Memory != nil {
				aggregated = append(aggregated, map[string]any{
					"timestamp":            entry.Timestamp,
					"memory_usage_percent": entry.Memory.UsedPercent,
				})
			}
		}
		return len(aggregated), aggregated
	case queryDisksReadSpeed:
		aggregated := make([]map[string]any, 0, n)
		for _, entry := range entries {
			if entry.DisksIO == nil {
				continue
			}
			m := make(map[string]any)
			for name, usage := range entry.DisksIO {
				m[name] = usage.ReadSpeed
			}
			m["timestamp"] = entry.Timestamp
			aggregated = append(aggregated, m)
		}
		return len(aggregated), aggregated
	case queryDisksWriteSpeed:
		aggregated := make([]map[string]any, 0, n)
		for _, entry := range entries {
			if entry.DisksIO == nil {
				continue
			}
			m := make(map[string]any)
			for name, usage := range entry.DisksIO {
				m[name] = usage.WriteSpeed
			}
			m["timestamp"] = entry.Timestamp
			aggregated = append(aggregated, m)
		}
		return len(aggregated), aggregated
	case queryDisksIOPS:
		aggregated := make([]map[string]any, 0, n)
		for _, entry := range entries {
			if entry.DisksIO == nil {
				continue
			}
			m := make(map[string]any)
			for name, usage := range entry.DisksIO {
				m[name] = usage.Iops
			}
			m["timestamp"] = entry.Timestamp
			aggregated = append(aggregated, m)
		}
		return len(aggregated), aggregated
	case queryDiskUsage:
		aggregated := make([]map[string]any, 0, n)
		for _, entry := range entries {
			if entry.Disks == nil {
				continue
			}
			m := make(map[string]any)
			for name, disk := range entry.Disks {
				m[name] = disk.Used
			}
			m["timestamp"] = entry.Timestamp
			aggregated = append(aggregated, m)
		}
		return len(aggregated), aggregated
	case queryNetworkSpeed:
		aggregated := make([]map[string]any, 0, n)
		for _, entry := range entries {
			if entry.Network == nil {
				continue
			}
			m := map[string]any{
				"timestamp": entry.Timestamp,
				"upload":    entry.Network.UploadSpeed,
				"download":  entry.Network.DownloadSpeed,
			}
			aggregated = append(aggregated, m)
		}
		return len(aggregated), aggregated
	case queryNetworkTransfer:
		aggregated := make([]map[string]any, 0, n)
		for _, entry := range entries {
			if entry.Network == nil {
				continue
			}
			m := map[string]any{
				"timestamp": entry.Timestamp,
				"upload":    entry.Network.BytesSent,
				"download":  entry.Network.BytesRecv,
			}
			aggregated = append(aggregated, m)
		}
		return len(aggregated), aggregated
	case querySensorTemperature:
		aggregated := make([]map[string]any, 0, n)
		for _, entry := range entries {
			if entry.Sensors == nil {
				continue
			}
			m := make(map[string]any)
			for key, sensor := range entry.Sensors {
				m[key] = sensor.Temperature
			}
			m["timestamp"] = entry.Timestamp
			aggregated = append(aggregated, m)
		}
		return len(aggregated), aggregated
	}
	return -1, []map[string]any{}
}
