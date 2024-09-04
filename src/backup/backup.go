package backup

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
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

type BackupManager struct {
	config     BackupConfig
	configFile string
	schedulers map[string]*time.Ticker
	mu         sync.Mutex
}

func NewBackupManager(configFile string) (*BackupManager, error) {
	bm := &BackupManager{
		configFile: configFile,
		schedulers: make(map[string]*time.Ticker),
	}
	err := bm.loadConfig()
	if err != nil {
		return nil, err
	}
	return bm, nil
}

func (bm *BackupManager) loadConfig() error {
	file, err := os.Open(bm.configFile)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	return decoder.Decode(&bm.config)
}

func (bm *BackupManager) StartBackupSchedule(mapName string) error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	config, ok := bm.config.Maps[mapName]
	if !ok {
		return fmt.Errorf("no configuration found for map: %s", mapName)
	}

	// Mark the map as having an active backup schedule
	saveFilePath := fmt.Sprintf("./data/%s.save", mapName)
	err := os.WriteFile(saveFilePath, []byte("true"), 0644)
	if err != nil {
		return fmt.Errorf("failed to write active schedule file: %w", err)
	}

	bm.startNewBackup(mapName, config)
	return nil
}

func (bm *BackupManager) resumeBackup(mapName string, config MapConfig, lastBackupFile string) {
	ticker := time.NewTicker(time.Duration(config.IntervalMinutes) * time.Minute)
	bm.schedulers[mapName] = ticker

	go func() {
		for range ticker.C {
			bm.IncrementalBackup(mapName, config)
		}
	}()
}

func (bm *BackupManager) startNewBackup(mapName string, config MapConfig) {
	ticker := time.NewTicker(time.Duration(config.IntervalMinutes) * time.Minute)
	bm.schedulers[mapName] = ticker

	go func() {
		bm.IncrementalBackup(mapName, config)
		for range ticker.C {
			bm.IncrementalBackup(mapName, config)
		}
	}()
}

func (bm *BackupManager) IncrementalBackup(mapName string, config MapConfig) error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	timestamp := time.Now().Format("20060102_150405")
	zipFileName := fmt.Sprintf("%s_%s.zip", mapName, timestamp)
	zipFilePath := filepath.Join(config.ZipDir, zipFileName)

	zipFile, err := os.Create(zipFilePath)
	if err != nil {
		return fmt.Errorf("failed to create zip file: %w", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	for _, ext := range config.FileExtensions {
		err := filepath.Walk(config.ExtractDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && filepath.Ext(info.Name()) == ext {
				err := bm.addFileToZip(zipWriter, path)
				if err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to add files with extension %s to zip: %w", ext, err)
		}
	}

	for _, file := range config.SpecificFiles {
		filePath := filepath.Join(config.ExtractDir, file)
		if _, err := os.Stat(filePath); err == nil {
			err := bm.addFileToZip(zipWriter, filePath)
			if err != nil {
				return fmt.Errorf("failed to add specific file %s to zip: %w", file, err)
			}
		}
	}

	lastBackupFile := fmt.Sprintf("./data/%s_saved.txt", mapName)
	err = os.WriteFile(lastBackupFile, []byte(timestamp), 0644)
	if err != nil {
		return fmt.Errorf("failed to write last backup timestamp: %w", err)
	}

	// Call RemoveOldBackups after creating the new backup
	err = bm.RemoveOldBackups(mapName, config)
	if err != nil {
		return fmt.Errorf("failed to remove old backups: %w", err)
	}

	return nil
}

func (bm *BackupManager) addFileToZip(zipWriter *zip.Writer, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	w, err := zipWriter.Create(filepath.Base(filePath))
	if err != nil {
		return fmt.Errorf("failed to create entry in zip file: %w", err)
	}

	_, err = io.Copy(w, file)
	if err != nil {
		return fmt.Errorf("failed to write file to zip: %w", err)
	}

	return nil
}

func (bm *BackupManager) RemoveOldBackups(mapName string, config MapConfig) error {
	retentionDuration := time.Duration(config.RetentionDays) * 24 * time.Hour
	now := time.Now()

	err := filepath.Walk(config.ZipDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(info.Name()) == ".zip" && info.ModTime().Add(retentionDuration).Before(now) {
			err := os.Remove(path)
			if err != nil {
				return fmt.Errorf("failed to remove old backup: %w", err)
			}
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to clean up old backups: %w", err)
	}

	return nil
}

func (bm *BackupManager) StopBackupSchedule(mapName string) error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	ticker, ok := bm.schedulers[mapName]
	if !ok {
		return fmt.Errorf("no running backup schedule for map: %s", mapName)
	}

	ticker.Stop()
	delete(bm.schedulers, mapName)

	// Mark the map as not having an active backup schedule
	saveFilePath := fmt.Sprintf("./data/%s.save", mapName)
	err := os.WriteFile(saveFilePath, []byte("false"), 0644)
	if err != nil {
		return fmt.Errorf("failed to write inactive schedule file: %w", err)
	}

	return nil
}

func (bm *BackupManager) StartOrResumeBackups() error {
	for mapName := range bm.config.Maps {
		saveFile := fmt.Sprintf("./data/%s.save", mapName) // Corrected path
		if _, err := os.Stat(saveFile); err == nil {
			data, err := os.ReadFile(saveFile)
			if err != nil {
				return fmt.Errorf("failed to read save file for %s: %w", mapName, err)
			}
			if string(data) == "true" {
				err := bm.StartBackupSchedule(mapName)
				if err != nil {
					return fmt.Errorf("failed to resume backup schedule for %s: %w", mapName, err)
				}
			}
		}
	}
	return nil
}
