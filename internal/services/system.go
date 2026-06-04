package services

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"
)

type SystemService struct {
	mainProcess *process.Process
}

func NewSystemService() *SystemService {
	p, _ := process.NewProcess(int32(os.Getpid()))
	return &SystemService{
		mainProcess: p,
	}
}

func (s *SystemService) GetMemoryInfo() map[string]interface{} {
	mainMem := uint64(0)
	if s.mainProcess != nil {
		if info, err := s.mainProcess.MemoryInfo(); err == nil {
			mainMem = info.RSS
		}
	}

	childrenMem := uint64(0)
	var childrenList []map[string]interface{}

	if s.mainProcess != nil {
		if children, err := s.mainProcess.Children(); err == nil {
			for _, child := range children {
				if name, err := child.Name(); err == nil {
					if info, err := child.MemoryInfo(); err == nil {
						childrenMem += info.RSS
						childrenList = append(childrenList, map[string]interface{}{
							"name":   name,
							"memory": info.RSS,
						})
					}
				}
			}
		}
	}

	return map[string]interface{}{
		"main_process":           mainMem,
		"children_processes":     childrenList,
		"total_children_memory":  childrenMem,
		"total_memory":           mainMem + childrenMem,
	}
}

func (s *SystemService) GetSystemStats() map[string]interface{} {
	cpuPercent, _ := cpu.Percent(100*time.Millisecond, false)
	vm, _ := mem.VirtualMemory()

	cpuVal := 0.0
	if len(cpuPercent) > 0 {
		cpuVal = cpuPercent[0]
	}

	vmMap := map[string]interface{}{
		"total":   0,
		"used":    0,
		"percent": 0,
	}
	if vm != nil {
		vmMap["total"] = vm.Total
		vmMap["used"] = vm.Used
		vmMap["percent"] = vm.UsedPercent
	}

	return map[string]interface{}{
		"cpu_percent":    cpuVal,
		"virtual_memory": vmMap,
		"os":             fmt.Sprintf("%s %s", runtime.GOOS, runtime.GOARCH),
		"go_version":     runtime.Version(),
	}
}

func (s *SystemService) GetWebInfo() map[string]interface{} {
	memInfo := s.GetMemoryInfo()
	stats := s.GetSystemStats()

	totalMem := memInfo["total_memory"].(uint64)
	totalMemMB := float64(totalMem) / 1024 / 1024

	return map[string]interface{}{
		"os":               stats["os"],
		"python_version":   stats["go_version"], // Keep key for web compat
		"memory_usage_mb": round(totalMemMB, 2),
		"cpu_percent":     stats["cpu_percent"],
	}
}

func (s *SystemService) FormatBytes(n uint64) string {
	nf := float64(n)
	units := []string{"B", "KB", "MB", "GB", "TB"}
	for _, unit := range units {
		if nf < 1024 {
			return fmt.Sprintf("%.2f %s", nf, unit)
		}
		nf /= 1024
	}
	return fmt.Sprintf("%.2f PB", nf)
}

func round(val float64, precision int) float64 {
	p := 1.0
	for i := 0; i < precision; i++ {
		p *= 10
	}
	return float64(int(val*p+0.5)) / p
}
