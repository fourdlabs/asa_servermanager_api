package backup

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// BackupConfig defines the configuration for backups
type BackupConfig struct {
	Maps map[string]MapConfig `json:"maps"`
}

type MapConfig struct {
	ZipDir          string   `json:"zip_dir"`
	ExtractDir      string   `json:"extract_dir"`
	FileExtensions  []string `json:"file_extensions"`
	SpecificFiles   []string `json:"specific_files"`
	IntervalMinutes int      `json:"interval_minutes"`
	RetentionDays   int      `json:"retention_days"`
}

// BackupManager manages backup schedules for multiple maps
type BackupManager struct {
	config      BackupConfig
	schedulers  map[string]*time.Ticker
	stopSignals map[string]chan struct{}
	mu          sync.Mutex
}

// NewBackupManager initializes a new BackupManager
func NewBackupManager(configFilePath string) (*BackupManager, error) {
	configFile, err := os.Open(configFilePath)
	if err != nil {
		return nil, err
	}
	defer configFile.Close()

	var config BackupConfig
	decoder := json.NewDecoder(configFile)
	err = decoder.Decode(&config)
	if err != nil {
		return nil, err
	}

	return &BackupManager{
		config:      config,
		schedulers:  make(map[string]*time.Ticker),
		stopSignals: make(map[string]chan struct{}),
	}, nil
}

// SaveLastBackupTime saves the last backup time to a file
func SaveLastBackupTime(mapName, timeStr string) error {
	filename := fmt.Sprintf("%s_saved.txt", mapName)
	return os.WriteFile(filename, []byte(timeStr), 0644)
}

// LoadLastBackupTime loads the last backup time from a file
func LoadLastBackupTime(mapName string) (string, error) {
	filename := fmt.Sprintf("%s_saved.txt", mapName)
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

// ZipFiles zips the files in the directory
func ZipFiles(srcDir, zipFilePath string, fileFilter func(string) bool) error {
	outFile, err := os.Create(zipFilePath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	zipWriter := zip.NewWriter(outFile)
	defer zipWriter.Close()

	err = filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		if !fileFilter(path) {
			return nil
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		zipFile, err := zipWriter.Create(relPath)
		if err != nil {
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(zipFile, file)
		return err
	})
	return err
}

// IncrementalBackup performs the incremental backup operation for a specific map
func IncrementalBackup(config MapConfig, mapName string) error {
	lastBackupTimeStr, err := LoadLastBackupTime(mapName)
	if err != nil {
		return err
	}

	var lastBackupTime time.Time
	if lastBackupTimeStr != "" {
		lastBackupTime, err = time.Parse(time.RFC3339, lastBackupTimeStr)
		if err != nil {
			return err
		}
	}

	// Define a filter function to select only files modified since the last backup
	fileFilter := func(path string) bool {
		fileInfo, err := os.Stat(path)
		if err != nil {
			return false
		}
		return fileInfo.ModTime().After(lastBackupTime)
	}

	currentTime := time.Now().Format(time.RFC3339)
	zipFilePath := filepath.Join(config.ZipDir, fmt.Sprintf("%s_backup_%s.zip", mapName, currentTime))
	err = ZipFiles(config.ExtractDir, zipFilePath, fileFilter)
	if err != nil {
		return err
	}

	// Save the timestamp of the last successful backup
	return SaveLastBackupTime(mapName, currentTime)
}

// RemoveOldBackups removes backups older than the retention period
func RemoveOldBackups(config MapConfig) error {
	retentionPeriod := time.Duration(config.RetentionDays) * 24 * time.Hour
	now := time.Now()

	files, err := os.ReadDir(config.ZipDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filePath := filepath.Join(config.ZipDir, file.Name())
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			return err
		}

		// Check if the file is older than the retention period
		if now.Sub(fileInfo.ModTime()) > retentionPeriod {
			err = os.Remove(filePath)
			if err != nil {
				return err
			}
			fmt.Printf("Removed old backup file: %s\n", filePath)
		}
	}

	return nil
}

// StartBackupSchedule starts or resumes backup scheduling for a specific map
func (bm *BackupManager) StartBackupSchedule(mapName string) error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	if _, exists := bm.schedulers[mapName]; exists {
		return fmt.Errorf("backup schedule for %s is already running", mapName)
	}

	mapConfig, exists := bm.config.Maps[mapName]
	if !exists {
		return fmt.Errorf("map %s not found in configuration", mapName)
	}

	ticker := time.NewTicker(time.Duration(mapConfig.IntervalMinutes) * time.Minute)
	stopSignal := make(chan struct{})

	bm.schedulers[mapName] = ticker
	bm.stopSignals[mapName] = stopSignal

	go func(name string, conf MapConfig) {
		// Perform initial backup
		err := IncrementalBackup(conf, name)
		if err != nil {
			log.Printf("Error backing up %s: %v\n", name, err)
		}

		// Remove old backups
		err = RemoveOldBackups(conf)
		if err != nil {
			log.Printf("Error removing old backups for %s: %v\n", name, err)
		}

		for {
			select {
			case <-ticker.C:
				err := IncrementalBackup(conf, name)
				if err != nil {
					log.Printf("Error backing up %s: %v\n", name, err)
				}
				err = RemoveOldBackups(conf)
				if err != nil {
					log.Printf("Error removing old backups for %s: %v\n", name, err)
				}
			case <-stopSignal:
				ticker.Stop()
				return
			}
		}
	}(mapName, mapConfig)

	return nil
}

// StopBackupSchedule stops the backup scheduling for a specific map
func (bm *BackupManager) StopBackupSchedule(mapName string) error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	ticker, exists := bm.schedulers[mapName]
	if !exists {
		return fmt.Errorf("backup schedule for %s is not running", mapName)
	}

	stopSignal, _ := bm.stopSignals[mapName]
	close(stopSignal)
	delete(bm.schedulers, mapName)
	delete(bm.stopSignals, mapName)

	return nil
}

// StartOrResumeBackups checks saved files and starts or resumes backup schedules
func (bm *BackupManager) StartOrResumeBackups() error {
	for mapName, mapConfig := range bm.config.Maps {
		lastBackupTimeStr, err := LoadLastBackupTime(mapName)
		if err != nil {
			return fmt.Errorf("failed to load last backup time for %s: %w", mapName, err)
		}

		if lastBackupTimeStr == "" {
			// If the saved file is empty, start the backup schedule
			err = bm.StartBackupSchedule(mapName)
			if err != nil {
				return fmt.Errorf("failed to start backup schedule for %s: %w", mapName, err)
			}
			fmt.Printf("Started backup schedule for %s\n", mapName)
		} else {
			// If the saved file has a timestamp, resume the backup schedule
			fmt.Printf("Resuming backup schedule for %s with last backup at %s\n", mapName, lastBackupTimeStr)
			// Optionally, start the schedule if it was not already running
			err = bm.StartBackupSchedule(mapName)
			if err != nil {
				return fmt.Errorf("failed to start backup schedule for %s: %w", mapName, err)
			}
		}
	}
	return nil
}
