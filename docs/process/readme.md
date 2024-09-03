# Process Manager

A simple Go library for managing multiple external processes, including starting, stopping, monitoring, and handling restarts. This library is compatible with Go v1.22+.

## Features

- Start and stop external processes with arguments.
- Monitor process status and restart processes if they stop.
- Handle process directories and store PID files.
- Enable or disable processes at runtime.
- Configurable restart intervals.

## Installation

To use the `processmanager` library in your Go project, follow these steps:

1. **Clone the repository** (replace `yourmodule` with the actual module path):

   ```bash
   git clone https://github.com/yourusername/processmanager.git
   ```

2. **Import the library** in your Go code:

   ```go
   import "yourmodule/processmanager"
   ```

3. **Install dependencies** (if any):

   ```bash
   go mod tidy
   ```

## Configuration

Create a JSON configuration file (`process_config.json`) with the following format:

```json
[
    {
        "map": "island",
        "dir": "./island_dir",
        "executable": "test1.exe",
        "args": ["--arg3", "value3", "--arg4", "value4"],
        "restart_interval": 5
    },
    {
        "map": "center",
        "dir": "./center_dir",
        "executable": "test2.exe",
        "args": ["--arg3", "value3", "--arg4", "value4"],
        "restart_interval": 5
    }
]
```

- `map`: Identifier for the process.
- `dir`: Directory to run the process in.
- `executable`: Path to the executable.
- `args`: Arguments to pass to the executable.
- `restart_interval`: Time (in seconds) to wait before restarting a stopped process.

## Usage

Here’s an example of how to use the `processmanager` library:

```go
package main

import (
    "log"
    "yourmodule/processmanager" // Replace with your actual module path
)

func main() {
    configFile := "process_config.json"

    pm, err := processmanager.NewProcessManager(configFile)
    if err != nil {
        log.Fatalf("Failed to create process manager: %v", err)
    }

    // Start all processes
    pm.StartAllProcesses()

    // Enable the 'island' process if it is not running
    pm.EnableProcess("island")

    // Disable the 'center' process if it is running
    pm.DisableProcess("center")

    // Prevent the main function from exiting immediately
    select {}
}
```

### API

- `NewProcessManager(configFile string) (*ProcessManager, error)`: Creates a new ProcessManager instance with the given configuration file.
- `StartAllProcesses()`: Starts all processes defined in the configuration.
- `EnableProcess(mapName string)`: Starts a specific process if it’s not already running.
- `DisableProcess(mapName string)`: Stops a specific process if it’s running.
