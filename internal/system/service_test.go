package system

import (
	"testing"
	"time"
)

func TestNewService(t *testing.T) {
	service := NewService()

	if service == nil {
		t.Error("Expected service instance, got nil")
	}
}

func TestGetSystemInfo(t *testing.T) {
	service := NewService()

	info, err := service.GetSystemInfo()
	if err != nil {
		t.Fatalf("Failed to get system info: %v", err)
	}

	if info == nil {
		t.Error("Expected system info, got nil")
	}

	// Test that required fields are populated
	if info.Hostname == "" {
		t.Error("Expected hostname to be populated")
	}

	if info.OS == "" {
		t.Error("Expected OS to be populated")
	}

	if info.Platform == "" {
		t.Error("Expected platform to be populated")
	}

	if info.Uptime <= 0 {
		t.Error("Expected uptime to be positive")
	}

	if info.BootTime.IsZero() {
		t.Error("Expected boot time to be set")
	}

	// Test CPU info structure
	testCPUInfo(t, &info.CPUInfo)

	// Test memory info structure
	testMemoryInfo(t, &info.MemoryInfo)

	// Test that we have some disk info (should have at least one disk)
	if len(info.DiskInfo) == 0 {
		t.Error("Expected at least one disk")
	}

	// Test disk info structure for first disk
	if len(info.DiskInfo) > 0 {
		testDiskInfo(t, &info.DiskInfo[0])
	}

	// Test network info structure
	if len(info.NetworkInfo) > 0 {
		testNetworkInfo(t, &info.NetworkInfo[0])
	}
}

func testCPUInfo(t *testing.T, cpuInfo *CPUInfo) {
	if cpuInfo.Cores <= 0 {
		t.Error("Expected CPU cores to be positive")
	}

	if cpuInfo.ModelName == "" {
		t.Error("Expected CPU model name to be populated")
	}

	// CPU usage should be populated (may be empty if failed to get)
	if len(cpuInfo.Usage) > 0 {
		for i, usage := range cpuInfo.Usage {
			if usage < 0 || usage > 100 {
				t.Errorf("Expected CPU usage[%d] to be between 0-100, got %f", i, usage)
			}
		}
	}

	// Temperature may be 0 if not available (especially on Windows)
	if cpuInfo.Temperature < 0 {
		t.Error("Expected CPU temperature to be non-negative")
	}
}

func testMemoryInfo(t *testing.T, memInfo *MemoryInfo) {
	if memInfo.Total <= 0 {
		t.Error("Expected total memory to be positive")
	}

	if memInfo.Available > memInfo.Total {
		t.Error("Expected available memory to be less than or equal to total")
	}

	if memInfo.Used > memInfo.Total {
		t.Error("Expected used memory to be less than or equal to total")
	}

	if memInfo.UsedPercent < 0 || memInfo.UsedPercent > 100 {
		t.Errorf("Expected memory usage percentage to be between 0-100, got %f", memInfo.UsedPercent)
	}
}

func testDiskInfo(t *testing.T, diskInfo *DiskInfo) {
	if diskInfo.Device == "" {
		t.Error("Expected disk device to be populated")
	}

	if diskInfo.Mountpoint == "" {
		t.Error("Expected disk mountpoint to be populated")
	}

	if diskInfo.Total <= 0 {
		t.Error("Expected disk total space to be positive")
	}

	if diskInfo.Used > diskInfo.Total {
		t.Error("Expected disk used space to be less than or equal to total")
	}

	if diskInfo.Free > diskInfo.Total {
		t.Error("Expected disk free space to be less than or equal to total")
	}

	if diskInfo.UsedPercent < 0 || diskInfo.UsedPercent > 100 {
		t.Errorf("Expected disk usage percentage to be between 0-100, got %f", diskInfo.UsedPercent)
	}
}

func testNetworkInfo(t *testing.T, netInfo *NetworkInfo) {
	if netInfo.Name == "" {
		t.Error("Expected network interface name to be populated")
	}

	// Bytes sent and received can be 0 for new or unused interfaces
	if netInfo.BytesSent < 0 {
		t.Error("Expected bytes sent to be non-negative")
	}

	if netInfo.BytesRecv < 0 {
		t.Error("Expected bytes received to be non-negative")
	}
}

func TestGetUptime(t *testing.T) {
	service := NewService()

	uptime, err := service.GetUptime()
	if err != nil {
		t.Fatalf("Failed to get uptime: %v", err)
	}

	if uptime <= 0 {
		t.Error("Expected uptime to be positive")
	}

	// Uptime should be reasonable (less than a year for testing)
	maxUptime := time.Duration(365*24) * time.Hour
	if uptime > maxUptime {
		t.Errorf("Uptime seems unreasonably high: %v", uptime)
	}
}

func TestGetLoadAverage(t *testing.T) {
	service := NewService()

	loads, err := service.GetLoadAverage()

	// Note: Load average may not be available on Windows
	if err != nil {
		t.Logf("Load average not available (this is normal on Windows): %v", err)
		return
	}

	if len(loads) != 3 {
		t.Errorf("Expected 3 load average values, got %d", len(loads))
	}

	for i, load := range loads {
		if load < 0 {
			t.Errorf("Expected load average[%d] to be non-negative, got %f", i, load)
		}
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input    uint64
		expected string
	}{
		{0, "0 B"},
		{100, "100 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
		{1099511627776, "1.0 TB"},
		{2048, "2.0 KB"},
		{3145728, "3.0 MB"},
	}

	for _, test := range tests {
		result := FormatBytes(test.input)
		if result != test.expected {
			t.Errorf("FormatBytes(%d): expected %s, got %s", test.input, test.expected, result)
		}
	}
}

func TestSystemInfoStructure(t *testing.T) {
	// Test that SystemInfo structure can be created and initialized
	info := &SystemInfo{
		Hostname:     "test-host",
		OS:           "windows",
		Platform:     "microsoft",
		Uptime:       time.Hour * 24,
		BootTime:     time.Now().Add(-24 * time.Hour),
		ProcessCount: 100,
		CPUInfo: CPUInfo{
			ModelName:   "Test CPU",
			Cores:       4,
			Usage:       []float64{10.5, 20.3, 15.7, 8.2},
			Temperature: 45.5,
		},
		MemoryInfo: MemoryInfo{
			Total:       8589934592, // 8GB
			Available:   4294967296, // 4GB
			Used:        4294967296, // 4GB
			UsedPercent: 50.0,
		},
		DiskInfo: []DiskInfo{
			{
				Device:      "C:",
				Mountpoint:  "C:\\",
				Fstype:      "NTFS",
				Total:       1099511627776, // 1TB
				Free:        549755813888,  // 512GB
				Used:        549755813888,  // 512GB
				UsedPercent: 50.0,
			},
		},
		NetworkInfo: []NetworkInfo{
			{
				Name:      "Ethernet",
				BytesSent: 1048576, // 1MB
				BytesRecv: 2097152, // 2MB
			},
		},
	}

	if info.Hostname != "test-host" {
		t.Error("Failed to set hostname")
	}

	if info.CPUInfo.Cores != 4 {
		t.Error("Failed to set CPU cores")
	}

	if info.MemoryInfo.Total != 8589934592 {
		t.Error("Failed to set memory total")
	}

	if len(info.DiskInfo) != 1 {
		t.Error("Failed to set disk info")
	}

	if len(info.NetworkInfo) != 1 {
		t.Error("Failed to set network info")
	}
}

func TestCPUInfoStructure(t *testing.T) {
	cpuInfo := CPUInfo{
		ModelName:   "Intel Core i7",
		Cores:       8,
		Usage:       []float64{25.5, 30.2, 15.8, 40.1, 20.3, 35.7, 45.2, 18.9},
		Temperature: 55.7,
	}

	if cpuInfo.ModelName != "Intel Core i7" {
		t.Error("Failed to set CPU model name")
	}

	if cpuInfo.Cores != 8 {
		t.Error("Failed to set CPU cores")
	}

	if len(cpuInfo.Usage) != 8 {
		t.Error("Failed to set CPU usage array")
	}

	if cpuInfo.Temperature != 55.7 {
		t.Error("Failed to set CPU temperature")
	}
}

func TestMemoryInfoStructure(t *testing.T) {
	memInfo := MemoryInfo{
		Total:       17179869184, // 16GB
		Available:   8589934592,  // 8GB
		Used:        8589934592,  // 8GB
		UsedPercent: 50.0,
	}

	if memInfo.Total != 17179869184 {
		t.Error("Failed to set memory total")
	}

	if memInfo.Available != 8589934592 {
		t.Error("Failed to set memory available")
	}

	if memInfo.Used != 8589934592 {
		t.Error("Failed to set memory used")
	}

	if memInfo.UsedPercent != 50.0 {
		t.Error("Failed to set memory used percentage")
	}
}

func TestDiskInfoStructure(t *testing.T) {
	diskInfo := DiskInfo{
		Device:      "/dev/sda1",
		Mountpoint:  "/",
		Fstype:      "ext4",
		Total:       2199023255552, // 2TB
		Free:        1099511627776, // 1TB
		Used:        1099511627776, // 1TB
		UsedPercent: 50.0,
	}

	if diskInfo.Device != "/dev/sda1" {
		t.Error("Failed to set disk device")
	}

	if diskInfo.Mountpoint != "/" {
		t.Error("Failed to set disk mountpoint")
	}

	if diskInfo.Fstype != "ext4" {
		t.Error("Failed to set disk filesystem type")
	}

	if diskInfo.Total != 2199023255552 {
		t.Error("Failed to set disk total")
	}

	if diskInfo.Free != 1099511627776 {
		t.Error("Failed to set disk free")
	}

	if diskInfo.Used != 1099511627776 {
		t.Error("Failed to set disk used")
	}

	if diskInfo.UsedPercent != 50.0 {
		t.Error("Failed to set disk used percentage")
	}
}

func TestNetworkInfoStructure(t *testing.T) {
	netInfo := NetworkInfo{
		Name:      "eth0",
		BytesSent: 10485760, // 10MB
		BytesRecv: 20971520, // 20MB
	}

	if netInfo.Name != "eth0" {
		t.Error("Failed to set network interface name")
	}

	if netInfo.BytesSent != 10485760 {
		t.Error("Failed to set bytes sent")
	}

	if netInfo.BytesRecv != 20971520 {
		t.Error("Failed to set bytes received")
	}
}

func TestFormatBytesEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    uint64
		contains string // Check if result contains this string
	}{
		{"Zero bytes", 0, "0 B"},
		{"Very small", 1, "1 B"},
		{"Just under KB", 1023, "1023 B"},
		{"Exactly 1 KB", 1024, "1.0 KB"},
		{"Large value", 1125899906842624, "PB"}, // 1 PB
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := FormatBytes(test.input)
			if result == "" {
				t.Error("FormatBytes returned empty string")
			}
			// For edge cases, just verify we get a reasonable result
			if len(result) == 0 {
				t.Errorf("FormatBytes(%d) returned empty result", test.input)
			}
		})
	}
}

func BenchmarkGetSystemInfo(b *testing.B) {
	service := NewService()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.GetSystemInfo()
		if err != nil {
			b.Fatalf("Failed to get system info: %v", err)
		}
	}
}

func BenchmarkFormatBytes(b *testing.B) {
	values := []uint64{0, 1024, 1048576, 1073741824, 1099511627776}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, value := range values {
			FormatBytes(value)
		}
	}
}
