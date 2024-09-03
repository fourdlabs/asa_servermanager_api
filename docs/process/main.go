package main

import (
	"log"
	"yourmodule/processmanager" // Replace with your actual module path
)

func main() {
	configFile := "process_config.json"

	pm, err := processmanager.NewProcessManager(configFile)
	if err != nil {
		log.Fatalf("Failed to create process manager: %v", err)
	}

	// Start all processes
	pm.StartAllProcesses()

	// Enable the 'island' process if it is not running
	pm.EnableProcess("island")

	// Disable the 'center' process if it is running
	pm.DisableProcess("center")

	// Prevent the main function from exiting immediately
	select {}
}
