Here's a `README.md` file for your backup library project. It includes sections on the project overview, installation, usage, and configuration.

### `README.md`

```markdown
# Backup Library

A Go library for managing and scheduling backups with support for multiple maps, incremental backups, retention policies, and zipping files. This library allows for individual control over backup schedules for each map.

## Features

- **Multiple Maps:** Support for different backup configurations for each map.
- **Incremental Backups:** Backup only the changes since the last backup.
- **Retention Policies:** Automatically remove old backups based on a retention period.
- **Zipping Files:** Compress files into ZIP archives for efficient storage.
- **Individual Scheduling:** Start and stop backup schedules independently for each map.

## Installation

1. **Clone the Repository**

   ```sh
   git clone https://github.com/yourusername/your-repository.git
   cd your-repository
   ```

2. **Build the Project**

   Ensure you have Go installed. Build the project using:

   ```sh
   go build -o backup-app main.go
   ```

3. **Dependencies**

   Make sure to add the library dependencies:

   ```sh
   go mod tidy
   ```

## Usage

1. **Create a Configuration File**

   Create a `config.json` file in the root directory with your backup configuration. For example:

   ```json
   {
       "maps": {
           "island": {
               "zip_dir": "backups/island",
               "extract_dir": "data/island",
               "file_extensions": [".txt", ".log"],
               "specific_files": ["important_file.txt"],
               "interval_minutes": 30,
               "retention_days": 7
           },
           "center": {
               "zip_dir": "backups/center",
               "extract_dir": "data/center",
               "file_extensions": [".csv", ".xml"],
               "specific_files": ["config.xml"],
               "interval_minutes": 60,
               "retention_days": 14
           }
       }
   }
   ```

2. **Run the Backup Application**

   To start the backup service, use:

   ```sh
   go run main.go
   ```

3. **Control Backup Schedules**

   You can start and stop backup schedules for specific maps programmatically. Here's an example of how to control the backup schedules:

   ```go
   package main

   import (
       "fmt"
       "log"
       "time"
       "your_project/backup" // Import the backup package
   )

   func main() {
       configFilePath := "config.json"
       
       // Create a new BackupManager
       manager, err := backup.NewBackupManager(configFilePath)
       if err != nil {
           log.Fatalf("Failed to initialize backup manager: %v", err)
       }

       // Start backup schedules for specific maps
       err = manager.StartBackupSchedule("island")
       if err != nil {
           log.Fatalf("Failed to start backup schedule for island: %v", err)
       }

       err = manager.StartBackupSchedule("center")
       if err != nil {
           log.Fatalf("Failed to start backup schedule for center: %v", err)
       }

       // Wait for a while before stopping a specific schedule
       time.Sleep(2 * time.Hour)

       // Stop the backup schedule for 'center'
       err = manager.StopBackupSchedule("center")
       if err != nil {
           log.Fatalf("Failed to stop backup schedule for center: %v", err)
       }

       fmt.Println("Backup schedules are running. Press Ctrl+C to exit.")
       select {} // Keep the application running
   }
   ```

## API Documentation

- **`BackupManager`**: Manages backup schedules.
  - `StartBackupSchedule(mapName string) error`: Starts a backup schedule for the specified map.
  - `StopBackupSchedule(mapName string) error`: Stops the backup schedule for the specified map.

- **Configuration Structure**:
  - `zip_dir`: Directory where the backup ZIP files will be stored.
  - `extract_dir`: Directory from which files will be backed up.
  - `file_extensions`: List of file extensions to include in the backup.
  - `specific_files`: List of specific files to include in the backup.
  - `interval_minutes`: How often the backup should occur for each map.
  - `retention_days`: How long to retain backups before deleting them.
