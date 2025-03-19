package hwmon

import (
	"fmt"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/host"
)

// GetCPUPercent retrieves CPU utilization percentage.
func GetCPUPercent() (float64, error) {
	percent, err := cpu.Percent(0, false)
	if err != nil {
		return 0, err
	}
	if len(percent) == 0 {
		return 0, fmt.Errorf("no CPU percent data")
	}
	return percent[0], nil
}

// GetChipsetTemp retrieves the chipset temperature.
func GetChipsetTemp() (float64, error) {
	sensors, err := host.SensorsTemperatures()
	if err != nil {
		return 0, err
	}
	for _, sensor := range sensors {
		if sensor.SensorKey == "k10temp" {
			return sensor.Temperature, nil
		}
	}
	return 0, fmt.Errorf("k10temp sensor not found")
}

// NewCPUMetric creates a new CPU utilization metric.
func NewCPUMetric() SendableMetric {
	return &AverageMetric{
		name:       "cpu_utilization_percent",
		sampleFunc: GetCPUPercent,
	}
}

// NewChipsetTempMetric creates a new chipset temperature metric.
func NewChipsetTempMetric() SendableMetric {
	return &AverageMetric{
		name:       "k10_temperature_celsius",
		sampleFunc: GetChipsetTemp,
	}
}