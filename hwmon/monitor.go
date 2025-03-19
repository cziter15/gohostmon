package hwmon

import (
	"fmt"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// HwMonitor manages hardware monitoring and MQTT publishing.
type HwMonitor struct {
	prefix           string
	client           mqtt.Client
	host             string
	user             string
	password         string
	updateInterval   time.Duration
	sendInterval     time.Duration
	updateMetrics    []UpdatableMetric
	sendMetrics      []SendableMetric
	lastMetricUpdate time.Time
	lastMetricSend   time.Time
}

// NewHwMonitor initializes a new hardware monitor.
func NewHwMonitor(mqttPrefix, host, user, password string, updateInterval, sendInterval time.Duration) *HwMonitor {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:1883", host))
	if user != "" {
		opts.SetUsername(user)
		opts.SetPassword(password)
	}
	client := mqtt.NewClient(opts)

	// Create metric instances
	metricsList := []SendableMetric{
		NewChipsetTempMetric(),
		NewCPUMetric(),
		NewMemoryMetric(),
		NewNetworkSentMetric(),
		NewNetworkReceivedMetric(),
	}

	// Separate metrics into update and send lists
	var updateMetrics []UpdatableMetric
	var sendMetrics []SendableMetric = metricsList
	for _, m := range metricsList {
		if um, ok := m.(UpdatableMetric); ok {
			updateMetrics = append(updateMetrics, um)
		}
	}

	return &HwMonitor{
		prefix:          mqttPrefix,
		client:          client,
		host:            host,
		user:            user,
		password:        password,
		updateInterval:  updateInterval,
		sendInterval:    sendInterval,
		updateMetrics:   updateMetrics,
		sendMetrics:     sendMetrics,
	}
}

// maybeUpdateMetrics updates metrics if the update interval has elapsed.
func (hm *HwMonitor) maybeUpdateMetrics() {
	now := time.Now()
	if now.Sub(hm.lastMetricUpdate) >= hm.updateInterval {
		for _, metric := range hm.updateMetrics {
			metric.Update()
		}
		hm.lastMetricUpdate = now
	}
}

// maybeSendMetrics sends metrics if the send interval has elapsed.
func (hm *HwMonitor) maybeSendMetrics() {
	now := time.Now()
	if now.Sub(hm.lastMetricSend) >= hm.sendInterval {
		for _, metric := range hm.sendMetrics {
			value, err := metric.GetValue()
			if err != nil {
				log.Printf("Error getting value for %s: %v", metric.Name(), err)
				continue
			}
			topic := hm.prefix + metric.Name()
			token := hm.client.Publish(topic, 0, false, value)
			token.Wait()
			if token.Error() != nil {
				log.Printf("Failed to publish to %s: %v", topic, token.Error())
			}
		}
		hm.lastMetricSend = now
	}
}

// Run starts the monitoring loop.
func (hm *HwMonitor) Run() {
	// Connect to MQTT with retry
	var token mqtt.Token
	for {
		token = hm.client.Connect()
		token.Wait()
		if token.Error() == nil {
			break
		}
		log.Printf("Failed to connect to MQTT broker: %v. Retrying in 5s", token.Error())
		time.Sleep(5 * time.Second)
	}

	// Initialize timers
	hm.lastMetricUpdate = time.Now()
	hm.lastMetricSend = time.Now()

	// Main loop
	for {
		hm.maybeUpdateMetrics()
		hm.maybeSendMetrics()
		time.Sleep(50 * time.Millisecond)
	}
}