package api

import (
	"asa_servermanager_api/backup"
	"asa_servermanager_api/processmanager"
	"asa_servermanager_api/rcon"
	"encoding/json"
	"log"
	"net/http"
)

var (
	process_conf = "config/process_config.json"
	backup_conf  = "config/backup_config.json"
)

func StartProcess(w http.ResponseWriter, r *http.Request) {
	mapName := r.URL.Query().Get("map")

	pm, err := processmanager.NewProcessManager(process_conf)
	if err != nil {
		log.Printf("Failed to create process manager: %v", err)
	}
	res := pm.EnableProcess(mapName)

	bm, err := backup.NewBackupManager(backup_conf)
	if err != nil {
		log.Printf("Failed to initialize BackupManager: %v", err)
	}

	err = bm.StartBackupSchedule(mapName)
	if err != nil {
		log.Printf("Failed to start backup schedule for map 'center': %v", err)
	}

	response := map[string]interface{}{
		"status": "Process started",
		"map":    mapName,
		"logs":   res,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func StopProcess(w http.ResponseWriter, r *http.Request) {
	mapName := r.URL.Query().Get("map")

	pm, err := processmanager.NewProcessManager(process_conf)
	if err != nil {
		log.Printf("Failed to create process manager: %v", err)
	}
	res := pm.DisableProcess(mapName)

	response := map[string]interface{}{
		"status": "Process started",
		"map":    mapName,
		"logs":   res,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func ListFiles(w http.ResponseWriter, r *http.Request) {
	mapName := r.URL.Query().Get("map")
	fileName := r.URL.Query().Get("file")

	log.Printf("Listing files %s in map %s", fileName, mapName)
	response := map[string][]string{"files": {"file1.zip", "file2.zip"}}
	json.NewEncoder(w).Encode(response)
}

func RestoreFile(w http.ResponseWriter, r *http.Request) {
	mapName := r.URL.Query().Get("map")
	zipName := r.URL.Query().Get("zip")
	fileName := r.URL.Query().Get("file")
	log.Printf("Restoring file %s from zip %s in map %s", fileName, zipName, mapName)
	response := map[string]string{"status": "File restored", "map": mapName, "file": fileName}
	json.NewEncoder(w).Encode(response)
}

func ManualBackup(w http.ResponseWriter, r *http.Request) {

	response := map[string]string{"status": "Manual backup initiated"}
	json.NewEncoder(w).Encode(response)
}

func ScheduleBackupOn(w http.ResponseWriter, r *http.Request) {
	mapName := r.URL.Query().Get("map")

	response := map[string]string{"status": "Scheduled backup on", "map": mapName}
	json.NewEncoder(w).Encode(response)
}

func ScheduleBackupOff(w http.ResponseWriter, r *http.Request) {
	mapName := r.URL.Query().Get("map")

	response := map[string]string{"status": "Scheduled backup off", "map": mapName}
	json.NewEncoder(w).Encode(response)
}

func RconComs(w http.ResponseWriter, r *http.Request) {
	mapName := r.URL.Query().Get("map")
	rComs := r.URL.Query().Get("command")
	repz := rcon.RconCommand(mapName, rComs)
	response := map[string]string{"status": "Command executed", "map": mapName, "data": repz}
	json.NewEncoder(w).Encode(response)
}

func GetMapLogs(w http.ResponseWriter, r *http.Request) {
	mapName := r.URL.Query().Get("map")

	logs, err := processmanager.RetrieveLogs(mapName)
	if err != nil {
		log.Printf("Failed to create process manager: %v", err)
	}

	response := map[string]interface{}{
		"status": "Logs retrieved",
		"map":    mapName,
		"logs":   logs,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
