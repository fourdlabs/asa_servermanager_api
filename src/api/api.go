package api

import (
	"asa_servermanager_api/backup"
	"asa_servermanager_api/processmanager"
	"log"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

var (
	limiter      = rate.NewLimiter(rate.Every(time.Second), 10)
	limiterMutex sync.Mutex
)

func rateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limiterMutex.Lock()
		defer limiterMutex.Unlock()

		if !limiter.Allow() {
			http.Error(w, "Rate limit exceeded. Try again later.", http.StatusTooManyRequests)
			return
		}
		next(w, r)
	}
}

func SetupRoutes() {

	process_conf := "config/process_config.json"
	pm, err := processmanager.NewProcessManager(process_conf)
	if err != nil {
		log.Fatalf("Failed to create process manager: %v", err)
	}
	pm.StartAllProcesses()

	backup_conf := "config/backup_config.json"
	bm, err := backup.NewBackupManager(backup_conf)
	if err != nil {
		log.Fatalf("Failed to initialize BackupManager: %v", err)
	}
	err = bm.StartOrResumeBackups()
	if err != nil {
		log.Fatalf("Failed to start or resume backups: %v", err)
	}

	http.HandleFunc("/start", rateLimitMiddleware(StartProcess))
	http.HandleFunc("/stop", rateLimitMiddleware(StopProcess))
	http.HandleFunc("/list", rateLimitMiddleware(ListFiles))
	http.HandleFunc("/restore", rateLimitMiddleware(RestoreFile))
	http.HandleFunc("/backup", rateLimitMiddleware(ManualBackup))
	http.HandleFunc("/backupon", rateLimitMiddleware(ScheduleBackupOn))
	http.HandleFunc("/backupoff", rateLimitMiddleware(ScheduleBackupOff))
	http.HandleFunc("/rcon", rateLimitMiddleware(RconComs))
	http.HandleFunc("/logs", rateLimitMiddleware(GetMapLogs))

	http.ListenAndServe(":8080", nil)
}
