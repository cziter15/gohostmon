package hwmon

import (
	"github.com/shirou/gopsutil/v3/mem"
)

// GetRAMPercent retrieves RAM usage percentage.
func GetRAMPercent() (float64, error) {
	vmem, err := mem.VirtualMemory()
	if err != nil {
		return 0, err
	}
	return vmem.UsedPercent, nil
}

// NewMemoryMetric creates a new memory usage metric.
func NewMemoryMetric() SendableMetric {
	return &AverageMetric{
		name:       "ram_used_percent",
		sampleFunc: GetRAMPercent,
	}
}