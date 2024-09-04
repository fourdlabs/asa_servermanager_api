package main

import (
	"asa_servermanager_api/api"
	"log"
	"os"
)

func main() {
	dataDir := "./data"
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		err := os.MkdirAll(dataDir, 0755)
		if err != nil {
			log.Printf("Failed to create data directory: %v", err)
		}
	}
	logFile := "./logs"
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		err := os.MkdirAll(logFile, 0755)
		if err != nil {
			log.Printf("Failed to create data directory: %v", err)
		}
	}
	api.SetupRoutes()
}
