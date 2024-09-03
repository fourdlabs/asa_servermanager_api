package processmanager

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"syscall"
	"time"
)

// ProcessConfig holds configuration for a single process.
type ProcessConfig struct {
	Map             string   `json:"map"`
	Dir             string   `json:"dir"`
	Executable      string   `json:"executable"`
	Args            []string `json:"args"`
	RestartInterval int      `json:"restart_interval"`
}

// ProcessManager manages the processes defined in the configuration.
type ProcessManager struct {
	configs   map[string]ProcessConfig
	processes map[string]*exec.Cmd
	mu        sync.Mutex
}

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

// StartProcess starts an external process with given command and arguments.
func StartProcess(executable string, args []string, dir string) (*exec.Cmd, error) {
	cmd := exec.Command(executable, args...)
	cmd.Dir = dir
	err := cmd.Start()
	if err != nil {
		return nil, err
	}
	return cmd, nil
}

// StopProcess terminates a running process.
func StopProcess(cmd *exec.Cmd) error {
	return cmd.Process.Kill()
}

// IsProcessRunning checks if a process with the given PID is running.
func IsProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// SavePID saves the PID to a file.
func SavePID(filename string, pid int) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = fmt.Fprintf(file, "%d", pid)
	return err
}

// ReadPID reads the PID from a file.
func ReadPID(filename string) (int, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return 0, err
	}
	return pid, nil
}

// GeneratePIDFileName generates a unique PID file name based on the process map name.
func GeneratePIDFileName(mapName string) string {
	return fmt.Sprintf("%s.pid", mapName)
}

// MonitorProcess monitors a process, restarts it if necessary, and handles PID management.
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
			log.Printf("Process '%s' has stopped or PID file is missing. Restarting...", mapName)
			cmd, err := StartProcess(config.Executable, config.Args, config.Dir)
			if err != nil {
				log.Printf("Failed to start process '%s': %v", mapName, err)
				time.Sleep(time.Duration(config.RestartInterval) * time.Second)
				continue
			}

			err = SavePID(pidFile, cmd.Process.Pid)
			if err != nil {
				log.Printf("Failed to save PID for process '%s': %v", mapName, err)
			}

			pm.mu.Lock()
			pm.processes[mapName] = cmd
			pm.mu.Unlock()

			cmd.Wait()
			os.Remove(pidFile)

			pm.mu.Lock()
			delete(pm.processes, mapName)
			pm.mu.Unlock()
		}

		time.Sleep(time.Duration(config.RestartInterval) * time.Second)
	}
}

// StartAllProcesses starts all processes defined in the configuration and begins monitoring them.
func (pm *ProcessManager) StartAllProcesses() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for _, config := range pm.configs {
		pidFile := GeneratePIDFileName(config.Map)
		if _, err := os.Stat(pidFile); err == nil {
			// PID file exists; check if the process is running
			pid, err := ReadPID(pidFile)
			if err == nil && IsProcessRunning(pid) {
				log.Printf("Resuming monitoring of existing process '%s' with PID %d", config.Map, pid)
				go pm.MonitorProcess(config.Map)
				continue
			}
			// PID file is present but process is not running; start new process
		}
		log.Printf("Starting process '%s'", config.Map)
		go pm.MonitorProcess(config.Map)
	}
}

// EnableProcess starts a specific process if it's not already running.
func (pm *ProcessManager) EnableProcess(mapName string) {
	pm.mu.Lock()
	config, exists := pm.configs[mapName]
	if exists {
		if _, running := pm.processes[mapName]; !running {
			go pm.MonitorProcess(mapName)
		}
	}
	pm.mu.Unlock()
}

// DisableProcess stops a specific process if it's running.
func (pm *ProcessManager) DisableProcess(mapName string) {
	pm.mu.Lock()
	cmd, running := pm.processes[mapName]
	if running {
		StopProcess(cmd)
		delete(pm.processes, mapName)
		os.Remove(GeneratePIDFileName(mapName))
	}
	pm.mu.Unlock()
}
