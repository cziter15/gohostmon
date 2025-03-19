package hwmon

import (
	"fmt"
	"log"
	"math"
	"strconv"
)

// UpdatableMetric defines a metric that requires periodic updates.
type UpdatableMetric interface {
	Update()
}

// SendableMetric defines a metric that can be collected and sent.
type SendableMetric interface {
	GetValue() (string, error)
	Name() string
}

// AverageMetric handles metrics that are sampled over time and averaged.
type AverageMetric struct {
	name       string
	sum        float64
	count      int
	sampleFunc func() (float64, error)
}

// Update samples the metric and accumulates the value.
func (m *AverageMetric) Update() {
	val, err := m.sampleFunc()
	if err == nil {
		m.sum += val
		m.count++
	} else {
		log.Printf("Error sampling %s: %v", m.name, err)
	}
}

// GetValue computes and returns the average, resetting the metric.
func (m *AverageMetric) GetValue() (string, error) {
	if m.count == 0 {
		return "", fmt.Errorf("no samples for %s", m.name)
	}
	avg := m.sum / float64(m.count)
	m.sum = 0
	m.count = 0
	return strconv.FormatFloat(math.Round(avg*10)/10, 'f', 2, 64), nil
}

// Name returns the metric's name.
func (m *AverageMetric) Name() string {
	return m.name
}

// RawMetric handles metrics collected directly when needed.
type RawMetric struct {
	name        string
	collectFunc func() (string, error)
}

// GetValue collects and returns the metric value.
func (m *RawMetric) GetValue() (string, error) {
	return m.collectFunc()
}

// Name returns the metric's name.
func (m *RawMetric) Name() string {
	return m.name
}