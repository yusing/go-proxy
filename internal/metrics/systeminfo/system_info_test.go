package systeminfo

import (
	"encoding/json"
	"testing"

	. "github.com/yusing/go-proxy/internal/utils/testing"
)

func TestSystemInfo(t *testing.T) {
	// Create test data
	cpuAvg := 45.67
	testInfo := &SystemInfo{
		Timestamp:  1234567890,
		CPUAverage: &cpuAvg,
		Memory: &MemoryUsage{
			Total:       16000000000,
			Available:   8000000000,
			Used:        8000000000,
			UsedPercent: 50.0,
		},
		Disks: map[string]*Disk{
			"sda": {
				Path:        "/",
				Fstype:      "ext4",
				Total:       500000000000,
				Free:        250000000000,
				Used:        250000000000,
				UsedPercent: 50.0,
			},
			"nvme0n1": {
				Path:        "/",
				Fstype:      "zfs",
				Total:       500000000000,
				Free:        250000000000,
				Used:        250000000000,
				UsedPercent: 50.0,
			},
		},
		DisksIO: map[string]*DiskIO{
			"media": {
				ReadBytes:  1000000,
				WriteBytes: 2000000,
				ReadSpeed:  100.5,
				WriteSpeed: 200.5,
				Iops:       1000,
			},
			"nvme0n1": {
				ReadBytes:  1000000,
				WriteBytes: 2000000,
				ReadSpeed:  100.5,
				WriteSpeed: 200.5,
				Iops:       1000,
			},
		},
		Network: &Network{
			BytesSent:     5000000,
			BytesRecv:     10000000,
			UploadSpeed:   1024.5,
			DownloadSpeed: 2048.5,
		},
		Sensors: map[string]Sensor{
			"cpu": {
				Temperature: 75.5,
				High:        85.0,
				Critical:    95.0,
			},
			"nvme0n1": {
				Temperature: 75.5,
				High:        85.0,
				Critical:    95.0,
			},
		},
	}

	// Test marshaling
	data, err := json.Marshal(testInfo)
	ExpectNoError(t, err)

	// Test unmarshaling back
	var decoded SystemInfo
	err = json.Unmarshal(data, &decoded)
	ExpectNoError(t, err)

	// Compare original and decoded
	ExpectEqual(t, decoded.Timestamp, testInfo.Timestamp)
	ExpectEqual(t, *decoded.CPUAverage, *testInfo.CPUAverage)
	ExpectDeepEqual(t, decoded.Memory, testInfo.Memory)
	ExpectDeepEqual(t, decoded.Disks, testInfo.Disks)
	ExpectDeepEqual(t, decoded.DisksIO, testInfo.DisksIO)
	ExpectDeepEqual(t, decoded.Network, testInfo.Network)
	ExpectDeepEqual(t, decoded.Sensors, testInfo.Sensors)

	// Test nil fields
	nilInfo := &SystemInfo{
		Timestamp: 1234567890,
	}

	data, err = json.Marshal(nilInfo)
	ExpectNoError(t, err)

	var decodedNil SystemInfo
	err = json.Unmarshal(data, &decodedNil)
	ExpectNoError(t, err)

	ExpectDeepEqual(t, decodedNil.Timestamp, nilInfo.Timestamp)
	ExpectTrue(t, decodedNil.CPUAverage == nil)
	ExpectTrue(t, decodedNil.Memory == nil)
	ExpectTrue(t, decodedNil.Disks == nil)
	ExpectTrue(t, decodedNil.Network == nil)
	ExpectTrue(t, decodedNil.Sensors == nil)
}
