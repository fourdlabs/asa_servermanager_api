### `main.go`

```go
// main.go
package main

import "yourmodule/api" // Replace 'yourmodule' with your module name

func main() {
    api.SetupRoutes()
}
```

### `api/api.go`

```go
// api/api.go
package api

import (
    "net/http"
    "golang.org/x/time/rate"
    "sync"
    "time"
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
```

### `api/handlers.go`

```go
// api/handlers.go
package api

import (
    "encoding/json"
    "net/http"
)

// StartProcess handles the /start endpoint
func StartProcess(w http.ResponseWriter, r *http.Request) {
    mapName := r.URL.Query().Get("map")
    // Placeholder for starting process logic
    response := map[string]string{"status": "Process started", "map": mapName}
    json.NewEncoder(w).Encode(response)
}

// StopProcess handles the /stop endpoint
func StopProcess(w http.ResponseWriter, r *http.Request) {
    mapName := r.URL.Query().Get("map")
    // Placeholder for stopping process logic
    response := map[string]string{"status": "Process stopped", "map": mapName}
    json.NewEncoder(w).Encode(response)
}

// ListFiles handles the /list endpoint
func ListFiles(w http.ResponseWriter, r *http.Request) {
    mapName := r.URL.Query().Get("map")
    fileName := r.URL.Query().Get("file")
    // Placeholder for listing files logic
    response := map[string][]string{"files": {"file1.zip", "file2.zip"}}
    json.NewEncoder(w).Encode(response)
}

// RestoreFile handles the /restore endpoint
func RestoreFile(w http.ResponseWriter, r *http.Request) {
    mapName := r.URL.Query().Get("map")
    zipName := r.URL.Query().Get("zip")
    fileName := r.URL.Query().Get("file")
    // Placeholder for restoring file logic
    response := map[string]string{"status": "File restored", "map": mapName, "file": fileName}
    json.NewEncoder(w).Encode(response)
}

// ManualBackup handles the /backup endpoint
func ManualBackup(w http.ResponseWriter, r *http.Request) {
    // Placeholder for manual backup logic
    response := map[string]string{"status": "Manual backup initiated"}
    json.NewEncoder(w).Encode(response)
}

// ScheduleBackupOn handles the /backupon endpoint
func ScheduleBackupOn(w http.ResponseWriter, r *http.Request) {
    mapName := r.URL.Query().Get("map")
    // Placeholder for scheduling backup logic
    response := map[string]string{"status": "Scheduled backup on", "map": mapName}
    json.NewEncoder(w).Encode(response)
}

// ScheduleBackupOff handles the /backupoff endpoint
func ScheduleBackupOff(w http.ResponseWriter, r *http.Request) {
    mapName := r.URL.Query().Get("map")
    // Placeholder for disabling scheduled backup logic
    response := map[string]string{"status": "Scheduled backup off", "map": mapName}
    json.NewEncoder(w).Encode(response)
}
```

### `README.md`

```markdown
# Go API with Rate Limiting

This project demonstrates a simple Go API with rate limiting applied to each endpoint. The API supports various operations including starting/stopping processes, listing files, restoring files from backups, and scheduling backups.

## Getting Started

### Prerequisites

- Go 1.22 or later
- `golang.org/x/time/rate` package

### Installation

1. **Clone the Repository**

   ```bash
   git clone <repository-url>
   cd <repository-directory>
   ```

2. **Install Dependencies**

   Install the `rate` package:

   ```bash
   go get golang.org/x/time/rate
   ```

### Running the Server

To start the server, run:

```bash
go run main.go
```

The server will start on port `8080`.

### API Endpoints

- **Start Process**
  - **Endpoint:** `/start`
  - **Method:** GET
  - **Query Parameter:** `map` (e.g., `/start?map=island`)

- **Stop Process**
  - **Endpoint:** `/stop`
  - **Method:** GET
  - **Query Parameter:** `map` (e.g., `/stop?map=island`)

- **List Files**
  - **Endpoint:** `/list`
  - **Method:** GET
  - **Query Parameters:** `map`, `file` (e.g., `/list?map=island&file=user.arkprofile`)

- **Restore File**
  - **Endpoint:** `/restore`
  - **Method:** GET
  - **Query Parameters:** `map`, `zip`, `file` (e.g., `/restore?map=island&zip=backup.zip&file=user.arkprofile`)

- **Manual Backup**
  - **Endpoint:** `/backup`
  - **Method:** GET

- **Schedule Backup On**
  - **Endpoint:** `/backupon`
  - **Method:** GET
  - **Query Parameter:** `map` (e.g., `/backupon?map=island`)

- **Schedule Backup Off**
  - **Endpoint:** `/backupoff`
  - **Method:** GET
  - **Query Parameter:** `map` (e.g., `/backupoff?map=island`)

### Rate Limiting

Each endpoint is rate limited to 1 request per second. If the rate limit is exceeded, the server responds with a `429 Too Many Requests` status.

