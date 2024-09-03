package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"yourmodule/backup" // Replace with the actual module path
)

func main() {
	// Initialize BackupManager with the path to your configuration file
	configFilePath := "backup_config.json"
	bm, err := backup.NewBackupManager(configFilePath)
	if err != nil {
		log.Fatalf("Failed to initialize BackupManager: %v", err)
	}

	// Start or resume backups based on the saved state
	err = bm.StartOrResumeBackups()
	if err != nil {
		log.Fatalf("Failed to start or resume backups: %v", err)
	}

	// Example: Start backup schedules for individual maps
	err = bm.StartBackupSchedule("island")
	if err != nil {
		log.Printf("Error starting backup schedule for island: %v", err)
	} else {
		fmt.Println("Backup schedule for island started.")
	}

	err = bm.StartBackupSchedule("center")
	if err != nil {
		log.Printf("Error starting backup schedule for center: %v", err)
	} else {
		fmt.Println("Backup schedule for center started.")
	}

	// Example: Wait for a while before stopping the backup schedules
	time.Sleep(5 * time.Minute)

	// Stop backup schedules for individual maps
	err = bm.StopBackupSchedule("island")
	if err != nil {
		log.Printf("Error stopping backup schedule for island: %v", err)
	} else {
		fmt.Println("Backup schedule for island stopped.")
	}

	err = bm.StopBackupSchedule("center")
	if err != nil {
		log.Printf("Error stopping backup schedule for center: %v", err)
	} else {
		fmt.Println("Backup schedule for center stopped.")
	}

	// Wait for user interrupt to gracefully shutdown the application
	fmt.Println("Press Ctrl+C to exit.")
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
}
