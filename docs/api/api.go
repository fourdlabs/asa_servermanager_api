// api/api.go
package api

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

var (
	// Create a rate limiter allowing 1 request per second
	limiter = rate.NewLimiter(rate.Every(time.Second), 1)

	// Mutex to synchronize access to the limiter
	limiterMutex sync.Mutex
)

// Rate limit middleware
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

// SetupRoutes sets up the HTTP routes and starts the server
func SetupRoutes() {
	http.HandleFunc("/start", rateLimitMiddleware(StartProcess))
	http.HandleFunc("/stop", rateLimitMiddleware(StopProcess))
	http.HandleFunc("/list", rateLimitMiddleware(ListFiles))
	http.HandleFunc("/restore", rateLimitMiddleware(RestoreFile))
	http.HandleFunc("/backup", rateLimitMiddleware(ManualBackup))
	http.HandleFunc("/backupon", rateLimitMiddleware(ScheduleBackupOn))
	http.HandleFunc("/backupoff", rateLimitMiddleware(ScheduleBackupOff))

	// Start the server
	http.ListenAndServe(":8080", nil)
}
