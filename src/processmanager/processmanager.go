package processmanager

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ProcessConfig holds configuration for a single process.
type ProcessConfig struct {
	Map             string   `json:"map"`
	Executable      string   `json:"executable"`
	Args            []string `json:"args"`
	RestartInterval int      `json:"restart_interval"`
}

// ProcessManager manages the processes defined in the configuration.
type ProcessManager struct {
	configs   map[string]ProcessConfig
	processes map[string]*exec.Cmd
	mu        sync.Mutex
	mapMu     sync.Mutex
}

var myMap = make(map[string]bool)
var myMapSarted = make(map[string]bool)

// NewProcessManager creates a new ProcessManager instance with the given configuration file.
func NewProcessManager(configFile string) (*ProcessManager, error) {
	pm := &ProcessManager{
		configs:   make(map[string]ProcessConfig),
		processes: make(map[string]*exec.Cmd),
	}

	configs, err := LoadProcessConfigs(configFile)
	if err != nil {
		return nil, err
	}

	for _, config := range configs {
		pm.configs[config.Map] = config
	}

	return pm, nil
}

// LoadProcessConfigs loads the process configurations from a JSON file.
func LoadProcessConfigs(filename string) ([]ProcessConfig, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var configs []ProcessConfig
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&configs); err != nil {
		return nil, err
	}
	return configs, nil
}

// StopProcess terminates a running process.
func StopProcess(cmd *exec.Cmd) error {
	return cmd.Process.Kill()
}

func IsProcessRunning(pid int) bool {
	// Convert PID to string
	pidStr := strconv.Itoa(pid)

	// Execute `tasklist` command to check if the process is running
	cmd := exec.Command("tasklist", "/FI", "PID eq "+pidStr)
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error executing tasklist command: %v", err)
		return false
	}

	// Check if PID appears in the output
	return strings.Contains(string(output), pidStr)
}

// SavePID saves the PID to a file.
func SavePID(filename string, pid int) error {
	// Ensure the directory exists
	dir := filepath.Dir(filename)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		log.Printf("Directory %s does not exist. Creating...", dir)
		if mkErr := os.MkdirAll(dir, 0755); mkErr != nil {
			return fmt.Errorf("failed to create directory %s: %v", dir, mkErr)
		}
	}

	// Create or open the PID file
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create PID file %s: %v", filename, err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			log.Printf("Failed to close PID file %s: %v", filename, closeErr)
		}
	}()

	// Write the PID to the file
	_, err = fmt.Fprintf(file, "%d", pid)
	if err != nil {
		return fmt.Errorf("failed to write PID to file %s: %v", filename, err)
	}

	log.Printf("PID %d saved to file %s", pid, filename)
	return nil
}

// ReadPID reads the PID from a file.
func ReadPID(filename string) (int, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return 0, fmt.Errorf("failed to read PID file %s: %v", filename, err)
	}
	var pid int
	_, err = fmt.Sscanf(string(data), "%d", &pid)
	if err != nil {
		return 0, fmt.Errorf("failed to parse PID from file %s: %v", filename, err)
	}
	return pid, nil
}

// RemovePID removes the PID file.
func RemovePID(filename string) error {
	return os.Remove(filename)
}

// GeneratePIDFileName generates a unique PID file name based on the process map name.
func GeneratePIDFileName(mapName string) string {
	return fmt.Sprintf("%s.pid", mapName)
}

func (pm *ProcessManager) MonitorProcess(mapName string) {
	pm.mu.Lock()
	config, exists := pm.configs[mapName]
	pm.mu.Unlock()

	if !exists {
		log.Printf("Process '%s' configuration not found. Skipping...", mapName)
		return
	}

	pidFile := GeneratePIDFileName(mapName)

	for {
		pid, err := ReadPID(pidFile)
		if err == nil && IsProcessRunning(pid) {
			log.Printf("Process '%s' is running with PID %d", mapName, pid)
		} else {
			if myMap[mapName] {
				myMapSarted[mapName] = true
				// Start the process
				cmd := exec.Command(config.Executable, config.Args...)
				cmd.Dir = filepath.Dir(config.Executable) // Set working directory to the executable's directory
				err = cmd.Start()
				if err != nil {
					log.Printf("Failed to start process '%s': %v", mapName, err)
					time.Sleep(time.Duration(config.RestartInterval) * time.Second)
					continue
				}

				// Attempt to save the PID to a file
				if err := SavePID(pidFile, cmd.Process.Pid); err != nil {
					log.Printf("Failed to save PID for process '%s': %v", mapName, err)
					// Kill the process if PID could not be saved
					cmd.Process.Kill()
					time.Sleep(time.Duration(config.RestartInterval) * time.Second)
					continue
				}

				log.Printf("Process '%s' started successfully with PID %d", mapName, cmd.Process.Pid)

				pm.mu.Lock()
				pm.processes[mapName] = cmd
				pm.mu.Unlock()

				// Wait for the process to exit and remove the PID file
				go func() {
					err := cmd.Wait()
					if err != nil {
						log.Printf("Process '%s' exited with error: %v", mapName, err)
					}
					if removeErr := RemovePID(pidFile); removeErr != nil {
						log.Printf("Failed to remove PID file for process '%s': %v", mapName, removeErr)
					}

					pm.mu.Lock()
					delete(pm.processes, mapName)
					pm.mu.Unlock()
				}()
			} else {
				log.Printf("Process '%s' is not enabled. Skipping...", mapName)
				break
			}

		}

		time.Sleep(time.Duration(config.RestartInterval) * time.Second)
	}
}

// StartAllProcesses resumes monitoring all processes defined in the configuration if a valid PID file exists.
func (pm *ProcessManager) StartAllProcesses() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for mapName := range pm.configs {
		pidFile := GeneratePIDFileName(mapName)
		if _, err := os.Stat(pidFile); err == nil {
			// PID file exists; check if the process is running
			pid, err := ReadPID(pidFile)
			if err == nil && IsProcessRunning(pid) {
				log.Printf("Resuming monitoring of existing process '%s' with PID %d", mapName, pid)
				myMap[mapName] = true
				go pm.MonitorProcess(mapName)
				continue
			}
		}
		// If PID file does not exist or process is not running, skip this process
		log.Printf("PID file for '%s' is missing or invalid. Skipping process...", mapName)
	}
}

// EnableProcess starts monitoring a specific process.
func (pm *ProcessManager) EnableProcess(mapName string) {
	myMap[mapName] = true
	pm.mu.Lock()
	if _, exists := pm.configs[mapName]; exists {
		if myMapSarted[mapName] {
			log.Printf("Map already running")
			return
		}
		go pm.MonitorProcess(mapName)
	}
	pm.mu.Unlock()
}

// DisableProcess stops monitoring a specific process.
func (pm *ProcessManager) DisableProcess(mapName string) {
	myMap[mapName] = false
	myMapSarted[mapName] = false
	pm.mu.Lock()
	cmd, running := pm.processes[mapName]
	if running {
		StopProcess(cmd)
		delete(pm.processes, mapName)
		RemovePID(GeneratePIDFileName(mapName))
	}
	pm.mu.Unlock()
}
