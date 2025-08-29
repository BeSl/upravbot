package system

import (
	"fmt"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

// SystemInfo содержит информацию о системе
type SystemInfo struct {
	Hostname     string        `json:"hostname"`
	OS           string        `json:"os"`
	Platform     string        `json:"platform"`
	Uptime       time.Duration `json:"uptime"`
	BootTime     time.Time     `json:"boot_time"`
	CPUInfo      CPUInfo       `json:"cpu_info"`
	MemoryInfo   MemoryInfo    `json:"memory_info"`
	DiskInfo     []DiskInfo    `json:"disk_info"`
	NetworkInfo  []NetworkInfo `json:"network_info"`
	ProcessCount uint64        `json:"process_count"`
}

type CPUInfo struct {
	ModelName   string    `json:"model_name"`
	Cores       int       `json:"cores"`
	Usage       []float64 `json:"usage"`
	Temperature float64   `json:"temperature"`
}

type MemoryInfo struct {
	Total       uint64  `json:"total"`
	Available   uint64  `json:"available"`
	Used        uint64  `json:"used"`
	UsedPercent float64 `json:"used_percent"`
}

type DiskInfo struct {
	Device      string  `json:"device"`
	Mountpoint  string  `json:"mountpoint"`
	Fstype      string  `json:"fstype"`
	Total       uint64  `json:"total"`
	Free        uint64  `json:"free"`
	Used        uint64  `json:"used"`
	UsedPercent float64 `json:"used_percent"`
}

type NetworkInfo struct {
	Name      string `json:"name"`
	BytesSent uint64 `json:"bytes_sent"`
	BytesRecv uint64 `json:"bytes_recv"`
}

// Service предоставляет методы для получения системной информации
type Service struct{}

// NewService создает новый экземпляр сервиса
func NewService() *Service {
	return &Service{}
}

// GetSystemInfo получает полную информацию о системе
func (s *Service) GetSystemInfo() (*SystemInfo, error) {
	info := &SystemInfo{}

	// Информация о хосте
	hostInfo, err := host.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to get host info: %w", err)
	}

	info.Hostname = hostInfo.Hostname
	info.OS = hostInfo.OS
	info.Platform = hostInfo.Platform
	info.Uptime = time.Duration(hostInfo.Uptime) * time.Second
	info.BootTime = time.Unix(int64(hostInfo.BootTime), 0)
	info.ProcessCount = hostInfo.Procs

	// Информация о CPU
	cpuInfo, err := s.getCPUInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU info: %w", err)
	}
	info.CPUInfo = *cpuInfo

	// Информация о памяти
	memInfo, err := s.getMemoryInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get memory info: %w", err)
	}
	info.MemoryInfo = *memInfo

	// Информация о дисках
	diskInfo, err := s.getDiskInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get disk info: %w", err)
	}
	info.DiskInfo = diskInfo

	// Информация о сети
	netInfo, err := s.getNetworkInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get network info: %w", err)
	}
	info.NetworkInfo = netInfo

	return info, nil
}

func (s *Service) getCPUInfo() (*CPUInfo, error) {
	cpuInfos, err := cpu.Info()
	if err != nil {
		return nil, err
	}

	info := &CPUInfo{
		Cores: runtime.NumCPU(),
	}

	if len(cpuInfos) > 0 {
		info.ModelName = cpuInfos[0].ModelName
	}

	// Получение загрузки CPU
	percentages, err := cpu.Percent(time.Second, true)
	if err == nil {
		info.Usage = percentages
	}

	// Попытка получить температуру (может не работать на Windows)
	temps, err := host.SensorsTemperatures()
	if err == nil && len(temps) > 0 {
		for _, temp := range temps {
			if temp.SensorKey == "cpu" {
				info.Temperature = temp.Temperature
				break
			}
		}
	}

	return info, nil
}

func (s *Service) getMemoryInfo() (*MemoryInfo, error) {
	memStat, err := mem.VirtualMemory()
	if err != nil {
		return nil, err
	}

	return &MemoryInfo{
		Total:       memStat.Total,
		Available:   memStat.Available,
		Used:        memStat.Used,
		UsedPercent: memStat.UsedPercent,
	}, nil
}

func (s *Service) getDiskInfo() ([]DiskInfo, error) {
	partitions, err := disk.Partitions(false)
	if err != nil {
		return nil, err
	}

	var diskInfos []DiskInfo
	for _, partition := range partitions {
		usage, err := disk.Usage(partition.Mountpoint)
		if err != nil {
			continue
		}

		diskInfos = append(diskInfos, DiskInfo{
			Device:      partition.Device,
			Mountpoint:  partition.Mountpoint,
			Fstype:      partition.Fstype,
			Total:       usage.Total,
			Free:        usage.Free,
			Used:        usage.Used,
			UsedPercent: usage.UsedPercent,
		})
	}

	return diskInfos, nil
}

func (s *Service) getNetworkInfo() ([]NetworkInfo, error) {
	netStats, err := net.IOCounters(true)
	if err != nil {
		return nil, err
	}

	var netInfos []NetworkInfo
	for _, stat := range netStats {
		netInfos = append(netInfos, NetworkInfo{
			Name:      stat.Name,
			BytesSent: stat.BytesSent,
			BytesRecv: stat.BytesRecv,
		})
	}

	return netInfos, nil
}

// GetUptime возвращает время работы системы
func (s *Service) GetUptime() (time.Duration, error) {
	hostInfo, err := host.Info()
	if err != nil {
		return 0, err
	}
	return time.Duration(hostInfo.Uptime) * time.Second, nil
}

// GetLoadAverage возвращает среднюю загрузку системы
func (s *Service) GetLoadAverage() ([]float64, error) {
	loadStat, err := load.Avg()
	if err != nil {
		return nil, err
	}
	return []float64{loadStat.Load1, loadStat.Load5, loadStat.Load15}, nil
}

// FormatBytes форматирует байты в читаемый формат
func FormatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
