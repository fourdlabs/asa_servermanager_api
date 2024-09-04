package processmanager

import (
	"bufio"
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

	"asa_servermanager_api/rcon"
)

type ProcessConfig struct {
	Map             string   `json:"map"`
	Executable      string   `json:"executable"`
	Args            []string `json:"args"`
	RestartInterval int      `json:"restart_interval"`
}

type ProcessManager struct {
	configs   map[string]ProcessConfig
	processes map[string]*exec.Cmd
	mu        sync.Mutex
}

var (
	myMap       = make(map[string]bool)
	myMapSarted = make(map[string]bool)
)

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

func IsProcessRunning(pid int) bool {

	pidStr := strconv.Itoa(pid)

	cmd := exec.Command("tasklist", "/FI", "PID eq "+pidStr)
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error executing tasklist command: %v", err)
		return false
	}

	return strings.Contains(string(output), pidStr)
}

func SavePID(filename string, pid int) error {
	dir := filepath.Dir(filename)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		log.Printf("Directory %s does not exist. Creating...", dir)
		if mkErr := os.MkdirAll(dir, 0755); mkErr != nil {
			return fmt.Errorf("failed to create directory %s: %v", dir, mkErr)
		}
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create PID file %s: %v", filename, err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			log.Printf("Failed to close PID file %s: %v", filename, closeErr)
		}
	}()

	_, err = fmt.Fprintf(file, "%d", pid)
	if err != nil {
		return fmt.Errorf("failed to write PID to file %s: %v", filename, err)
	}

	log.Printf("PID %d saved to file %s", pid, filename)
	return nil
}

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

func RemovePID(filename string) error {
	return os.Remove(filename)
}

func GeneratePIDFileName(mapName string) string {
	return fmt.Sprintf("./data/%s.pid", mapName)
}

func (pm *ProcessManager) MonitorProcess(mapName string) {
	pm.mu.Lock()
	config, exists := pm.configs[mapName]
	pm.mu.Unlock()

	if !exists {
		log.Printf("Process '%s' configuration not found. Skipping...", mapName)
		return
	}

	pidFile := filepath.Join("./data", GeneratePIDFileName(mapName))
	logFile, err := CreateLogFile(mapName)
	if err != nil {
		log.Printf("Error creating log file: %v", err)
		return
	}
	defer logFile.Close()

	for {
		pid, err := ReadPID(pidFile)
		if err == nil && IsProcessRunning(pid) {

			time.Sleep(time.Duration(config.RestartInterval) * time.Second)
			continue
		}

		if myMap[mapName] {
			myMap[mapName] = true
			myMapSarted[mapName] = true

			cmd := exec.Command(config.Executable, config.Args...)
			cmd.Dir = filepath.Dir(config.Executable)

			stdoutPipe, err := cmd.StdoutPipe()
			if err != nil {
				log.Printf("Failed to create stdout pipe for process '%s': %v", mapName, err)
				time.Sleep(time.Duration(config.RestartInterval) * time.Second)
				continue
			}
			stderrPipe, err := cmd.StderrPipe()
			if err != nil {
				log.Printf("Failed to create stderr pipe for process '%s': %v", mapName, err)
				time.Sleep(time.Duration(config.RestartInterval) * time.Second)
				continue
			}

			if err := cmd.Start(); err != nil {
				log.Printf("Failed to start process '%s': %v", mapName, err)
				time.Sleep(time.Duration(config.RestartInterval) * time.Second)
				continue
			}

			go func() {
				scanner := bufio.NewScanner(stdoutPipe)
				for scanner.Scan() {
					logMessage := fmt.Sprintf("%s", scanner.Text())
					WriteLog(logFile, logMessage)
				}
			}()
			go func() {
				scanner := bufio.NewScanner(stderrPipe)
				for scanner.Scan() {
					logMessage := fmt.Sprintf("%s", scanner.Text())
					WriteLog(logFile, logMessage)
				}
			}()

			if err := SavePID(pidFile, cmd.Process.Pid); err != nil {
				log.Printf("Failed to save PID for process '%s': %v", mapName, err)
				cmd.Process.Kill()
				time.Sleep(time.Duration(config.RestartInterval) * time.Second)
				continue
			}

			log.Printf("Process '%s' started successfully with PID %d", mapName, cmd.Process.Pid)

			pm.mu.Lock()
			pm.processes[mapName] = cmd
			pm.mu.Unlock()

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

		time.Sleep(time.Duration(config.RestartInterval) * time.Second)
	}
}

func CreateLogFile(mapName string) (*os.File, error) {

	dateStr := time.Now().Format("01-02-2006")
	timeStr := time.Now().Format("03_04_PM")
	logFileName := fmt.Sprintf("./logs/%s_%s_%s.log", mapName, dateStr, timeStr)

	file, err := os.Create(logFileName)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file %s: %v", logFileName, err)
	}
	return file, nil
}

func WriteLog(file *os.File, message string) error {
	_, err := file.WriteString(message + "\n")
	if err != nil {
		return fmt.Errorf("failed to write to log file: %v", err)
	}
	return nil
}

func RetrieveLogs(mapName string) (string, error) {

	dateStr := time.Now().Format("01-02-2006")
	logFileName := fmt.Sprintf("./logs/%s_%s.log", mapName, dateStr)

	file, err := os.Open(logFileName)
	if err != nil {
		if os.IsNotExist(err) {
			return "No logs found for the specified process.", nil
		}
		return "", fmt.Errorf("failed to open log file %s: %w", logFileName, err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("failed to stat log file %s: %w", logFileName, err)
	}

	if stat.Size() == 0 {
		return "Log file is empty.", nil
	}

	data := make([]byte, stat.Size())
	_, err = file.Read(data)
	if err != nil {
		return "", fmt.Errorf("failed to read log file %s: %w", logFileName, err)
	}

	return string(data), nil
}

func (pm *ProcessManager) StartAllProcesses() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for mapName := range pm.configs {
		pidFile := GeneratePIDFileName(mapName)
		if _, err := os.Stat(pidFile); err == nil {

			pid, err := ReadPID(pidFile)
			if err == nil && IsProcessRunning(pid) {
				log.Printf("Resuming monitoring of existing process '%s' with PID %d", mapName, pid)
				myMap[mapName] = true
				go pm.MonitorProcess(mapName)
				continue
			}
		}

		log.Printf("PID file for '%s' is missing or invalid. Skipping process...", mapName)
	}
}

func (pm *ProcessManager) EnableProcess(mapName string) string {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if _, exists := pm.configs[mapName]; exists {
		if myMapSarted[mapName] {
			log.Printf("Map already running")
			return "Map already running"
		}
		myMap[mapName] = true
		go pm.MonitorProcess(mapName)
		return "Successfully started the map " + mapName
	}

	return "Eror: Map " + mapName + " not found"
}

func mergedID(m string, e string) string {
	return fmt.Sprintf("%s%s", m, e)
}

func (pm *ProcessManager) DisableProcess(mapName string) string {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	myMap[mapName] = false
	myMapSarted[mapName] = false

	if rcon.DummyRcon(mapName, "doexit") == "Exiting... \n " {
		delete(pm.processes, mapName)
		RemovePID(mergedID(mapName, "_saved.pid"))
		RemovePID(mergedID(mapName, ".save"))
		return "Successfully stopped the map " + mapName
	}

	return "Error: Shutting down the map " + mapName
}
