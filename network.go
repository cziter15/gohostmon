package hwmon

import (
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/v3/net"
)

// isPhysicalEthernetInterface checks if a network interface is a physical Ethernet device.
func isPhysicalEthernetInterface(name string) bool {
	return strings.HasPrefix(name, "eth") || strings.HasPrefix(name, "enp")
}

// GetNetworkBytesSent retrieves total bytes sent over physical network interfaces.
func GetNetworkBytesSent() (uint64, error) {
	netStats, err := net.IOCounters(true)
	if err != nil {
		return 0, err
	}
	var totalBytesSent uint64
	for _, stat := range netStats {
		if isPhysicalEthernetInterface(stat.Name) {
			totalBytesSent += stat.BytesSent
		}
	}
	return totalBytesSent, nil
}

// GetNetworkBytesReceived retrieves total bytes received over physical network interfaces.
func GetNetworkBytesReceived() (uint64, error) {
	netStats, err := net.IOCounters(true)
	if err != nil {
		return 0, err
	}
	var totalBytesReceived uint64
	for _, stat := range netStats {
		if isPhysicalEthernetInterface(stat.Name) {
			totalBytesReceived += stat.BytesRecv
		}
	}
	return totalBytesReceived, nil
}

// NewNetworkSentMetric creates a new network bytes sent metric.
func NewNetworkSentMetric() SendableMetric {
	return &RawMetric{
		name: "network_total_bytes_sent",
		collectFunc: func() (string, error) {
			bytes, err := GetNetworkBytesSent()
			if err != nil {
				return "", err
			}
			return strconv.FormatUint(bytes, 10), nil
		},
	}
}

// NewNetworkReceivedMetric creates a new network bytes received metric.
func NewNetworkReceivedMetric() SendableMetric {
	return &RawMetric{
		name: "network_total_bytes_received",
		collectFunc: func() (string, error) {
			bytes, err := GetNetworkBytesReceived()
			if err != nil {
				return "", err
			}
			return strconv.FormatUint(bytes, 10), nil
		},
	}
}