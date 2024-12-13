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
	lastBytesSent     uint64
	lastBytesReceived uint64
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
	sensors, _ := host.SensorsTemperatures()
	if len(sensors) == 0 {
		return 0
	}
	for _, sensor := range sensors {
		if sensor.SensorKey == "k10temp" {
			return sensor.Temperature
		}
	}
	return 0
}

func (hm *HwMonitor) getNetworkMetrics() (uint64, uint64) {
	netStats, err := net.IOCounters(true) // true fetches stats per interface
	if err != nil || len(netStats) == 0 {
		return 0, 0
	}

	var totalBytesSent, totalBytesReceived uint64
	for _, stat := range netStats {
		if isPhysicalEthernetInterface(stat.Name) {
			totalBytesSent += stat.BytesSent
			totalBytesReceived += stat.BytesRecv
		}
	}

	return totalBytesSent, totalBytesReceived
}

func isPhysicalEthernetInterface(name string) bool {
	// Exclude loopback interfaces
	if strings.HasPrefix(name, "lo") {
		return false
	}

	// Exclude Docker-related interfaces
	if strings.HasPrefix(name, "docker") ||
		strings.HasPrefix(name, "br-") ||
		strings.HasPrefix(name, "veth") ||
		strings.HasPrefix(name, "tun") || // Often used for VPNs/tunnels
		strings.HasPrefix(name, "virbr") || // Virtual bridges
		strings.HasPrefix(name, "kube") { // Kubernetes-related
		return false
	}

	// Include only physical Ethernet interfaces (e.g., "eth0", "enp3s0")
	return strings.HasPrefix(name, "eth") || strings.HasPrefix(name, "enp")
}

func (hm *HwMonitor) collectMetric(key string, value float64) {
	hm.metricValues[key] += value
}

func (hm *HwMonitor) maybeUpdateMetrics() {
	now := time.Now()
	if now.Sub(hm.lastMetricUpdate) >= hm.updateInterval {
		// Read temperatures.
		hm.collectMetric("k10_temperature_celsius", hm.getChipsetTemp())

		// Handle CPU utilization.
		cpuPercent, _ := cpu.Percent(0, false)
		if len(cpuPercent) > 0 {
			hm.collectMetric("cpu_utilization_percent", cpuPercent[0])
		}

		// Read VM stats.
		vmem, _ := mem.VirtualMemory()
		hm.collectMetric("ram_used_percent", vmem.UsedPercent)

		// Update counter and timer.
		hm.updateCounter++
		hm.lastMetricUpdate = now
	}
}

func (hm *HwMonitor) maybeSendMetrics() {
	// Handle timer.
	now := time.Now()
	if now.Sub(hm.lastMetricSend) >= hm.sendInterval {
		// Collect network metrics.
		totalSent, totalReceived := hm.getNetworkMetrics()
		hm.collectMetric("network_total_bytes_sent", totalSent)
		hm.collectMetric("network_total_bytes_received", totalReceived)

		// Collect average-based metrics.
		for key, value := range hm.metricValues {
			val := value / float64(hm.updateCounter)
			topic := hm.prefix + key
			hm.client.Publish(topic, 0, false, strconv.FormatFloat(math.Round(val*10)/10, 'f', HWMON_ROUNDING_PRECISION, 64))
			hm.metricValues[key] = 0
		}

		// Reset counter and timer.
		hm.updateCounter = 0
		hm.lastMetricSend = now
	}
}

func (hm *HwMonitor) run() {
	// Spawn MQTT client.
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

	// Init timers.
	hm.lastMetricSend = time.Now()
	hm.lastMetricUpdate = time.Now()
	
	// Core loop logic.
	for {
		hm.maybeUpdateMetrics()
		hm.maybeSendMetrics()
		time.Sleep(50 * time.Millisecond)
	}
}

func main() {
	fmt.Println("[HWMON starting]")

	// Read config.
	cfg, err := ini.Load("config.ini")
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}
	hostname := cfg.Section("credentials").Key("host").String()
	username := cfg.Section("credentials").Key("user").String()
	password := cfg.Section("credentials").Key("pass").String()

	// Print config.
	fmt.Println("> MQTT hostname is", hostname)
	fmt.Println("> MQTT user is", username)

	// Spawn the tool and run it forever.
	monitor := NewHwMonitor(HWMON_MQTT_PREFIX, hostname, username, password, HWMON_METRIC_UPDATE_INTERVAL, HWMON_METRIC_SEND_INTERVAL)
	fmt.Println("[Monitor starting]")
	monitor.run()
}
