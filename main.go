package main

import (
	"fmt"
	"log"
	"math"
	"strconv"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"gopkg.in/ini.v1"
)

const (
	HWMON_MQTT_PREFIX            = "hwinfo/"
	HWMON_MQTT_PORT              = 1883
	HWMON_METRIC_UPDATE_INTERVAL = 1 * time.Second
	HWMON_METRIC_SEND_INTERVAL   = 30 * time.Second
	HWMON_KEEPALIVE_INTERVAL     = 60
	HWMON_ROUNDING_PRECISION     = 1
)

type HwMonitor struct {
	prefix            string
	client            mqtt.Client
	host              string
	user              string
	password          string
	updateInterval    time.Duration
	sendInterval      time.Duration
	updateCounter     int
	metricValues      map[string]float64
	lastMetricSend    time.Time
	lastMetricUpdate  time.Time
	lastBytesSent     float64
	lastBytesReceived float64
}

func NewHwMonitor(mqttPrefix, host, user, password string, updateInterval, sendInterval time.Duration) *HwMonitor {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", host, HWMON_MQTT_PORT))
	if user != "" {
		opts.SetUsername(user)
		opts.SetPassword(password)
	}
	client := mqtt.NewClient(opts)

	return &HwMonitor{
		prefix:         mqttPrefix,
		client:         client,
		host:           host,
		user:           user,
		password:       password,
		updateInterval: updateInterval,
		sendInterval:   sendInterval,
		metricValues:   make(map[string]float64),
	}
}

func (hm *HwMonitor) getChipsetTemp() float64 {
	sensors, err := host.SensorsTemperatures()
	if err != nil || len(sensors) == 0 {
		return 0
	}
	for _, sensor := range sensors {
		if sensor.SensorKey == "k10temp" {
			return sensor.Temperature
		}
	}
	return 0
}

func (hm *HwMonitor) getNetworkMetrics() (float64, float64) {
	netStats, err := net.IOCounters(false)
	if err != nil || len(netStats) == 0 {
		return 0, 0
	}

	var totalBytesSent, totalBytesReceived float64
	for _, stat := range netStats {
		totalBytesSent += float64(stat.BytesSent)
		totalBytesReceived += float64(stat.BytesRecv)
	}

	// Calculate the bytes per second (Bps)
	bytesSentPerSec := totalBytesSent - hm.lastBytesSent
	bytesReceivedPerSec := totalBytesReceived - hm.lastBytesReceived

	// Store current values for next interval calculation
	hm.lastBytesSent = totalBytesSent
	hm.lastBytesReceived = totalBytesReceived

	// Convert bytes per second to Mbit per second (1 byte = 8 bits, 1 Mbit = 1,000,000 bits)
	mbpsSent := (bytesSentPerSec * 8) / 1000000
	mbpsReceived := (bytesReceivedPerSec * 8) / 1000000

	return mbpsSent, mbpsReceived
}

func (hm *HwMonitor) collectMetric(key string, value float64) {
	hm.metricValues[key] += value
}

func (hm *HwMonitor) maybeUpdateMetrics() {
	now := time.Now()
	if now.Sub(hm.lastMetricUpdate) >= hm.updateInterval {
		hm.collectMetric("k10_temperature_celsius", hm.getChipsetTemp())

		cpuPercent, _ := cpu.Percent(0, false)
		if len(cpuPercent) > 0 {
			hm.collectMetric("cpu_utilization_percent", cpuPercent[0])
		}

		vmem, _ := mem.VirtualMemory()
		hm.collectMetric("ram_used_percent", vmem.UsedPercent)

		// Collect network metrics
		mbpsSent, mbpsReceived := hm.getNetworkMetrics()
		hm.collectMetric("network_mbps_sent", mbpsSent)
		hm.collectMetric("network_mbps_received", mbpsReceived)

		hm.updateCounter++
		hm.lastMetricUpdate = now
	}
}

func (hm *HwMonitor) maybeSendMetrics() {
	now := time.Now()
	if now.Sub(hm.lastMetricSend) >= hm.sendInterval {
		for key, value := range hm.metricValues {
			val := value / float64(hm.updateCounter)
			topic := hm.prefix + key
			hm.client.Publish(topic, 0, false, strconv.FormatFloat(math.Round(val*10)/10, 'f', HWMON_ROUNDING_PRECISION, 64))
			hm.metricValues[key] = 0
		}
		hm.updateCounter = 0
		hm.lastMetricSend = now
	}
}

func (hm *HwMonitor) run() {
	if token := hm.client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Failed to connect to MQTT broker: %v", token.Error())
	}
	hm.lastMetricSend = time.Now()
	hm.lastMetricUpdate = time.Now()

	for {
		hm.maybeUpdateMetrics()
		hm.maybeSendMetrics()
		time.Sleep(50 * time.Millisecond)
	}
}

func main() {
	fmt.Println("[HWMON starting]")

	cfg, err := ini.Load("config.ini")
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}

	hostname := cfg.Section("credentials").Key("host").String()
	username := cfg.Section("credentials").Key("user").String()
	password := cfg.Section("credentials").Key("pass").String()

	fmt.Println("> MQTT hostname is", hostname)
	fmt.Println("> MQTT user is", username)

	monitor := NewHwMonitor(HWMON_MQTT_PREFIX, hostname, username, password, HWMON_METRIC_UPDATE_INTERVAL, HWMON_METRIC_SEND_INTERVAL)
	fmt.Println("[Monitor starting]")
	monitor.run()
}
