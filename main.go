package main

import (
	"log"
	"time"

	"hwmon"
	"gopkg.in/ini.v1"
)

func main() {
	log.Println("Loading configuration file")

	// Load configuration
	cfg, err := ini.Load("config.ini")
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}
	hostname := cfg.Section("credentials").Key("host").String()
	username := cfg.Section("credentials").Key("user").String()
	password := cfg.Section("credentials").Key("pass").String()

	// Print configuration
	log.Println("Configured MQTT hostname is", hostname)
	log.Println("Configured MQTT user is", username)

	// Create and run the monitor
	monitor := hwmon.NewHwMonitor(
		"hwinfo/",
		hostname,
		username,
		password,
		1*time.Second,  // Update interval
		30*time.Second, // Send interval
	)

	log.Println("Starting hardware monitor")

	monitor.Run()
}