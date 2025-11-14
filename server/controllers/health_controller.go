package controllers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"syscall"
	"time"
	"unsafe"
)

// HealthStatus represents the overall health status
type HealthStatus struct {
	Status    string            `json:"status"`     // "healthy", "degraded", "unhealthy"
	Timestamp string            `json:"timestamp"`  // ISO 8601 format
	Version   string            `json:"version"`    // App version
	Uptime    string            `json:"uptime"`     // Server uptime
	Checks    map[string]Check  `json:"checks"`     // Individual health checks
}

// Check represents a single health check result
type Check struct {
	Status  string `json:"status"`           // "pass", "fail", "warn"
	Message string `json:"message,omitempty"` // Additional info
	Time    string `json:"time,omitempty"`    // Response time in ms
}

var startTime = time.Now()

// HealthCheck handles GET /health
func HealthCheck(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		health := HealthStatus{
			Status:    "healthy",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Version:   getVersion(),
			Uptime:    getUptime(),
			Checks:    make(map[string]Check),
		}

		// Check database connectivity
		dbCheck := checkDatabase(db)
		health.Checks["database"] = dbCheck
		if dbCheck.Status == "fail" {
			health.Status = "unhealthy"
		}

		// Check disk space
		diskCheck := checkDiskSpace()
		health.Checks["disk"] = diskCheck
		if diskCheck.Status == "fail" {
			health.Status = "unhealthy"
		} else if diskCheck.Status == "warn" && health.Status == "healthy" {
			health.Status = "degraded"
		}

		// Check memory usage
		memCheck := checkMemory()
		health.Checks["memory"] = memCheck
		if memCheck.Status == "warn" && health.Status == "healthy" {
			health.Status = "degraded"
		}

		// Set HTTP status code based on health
		statusCode := http.StatusOK
		if health.Status == "unhealthy" {
			statusCode = http.StatusServiceUnavailable
		} else if health.Status == "degraded" {
			statusCode = http.StatusOK // Still operational
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(health)
	}
}

// checkDatabase verifies database connectivity
func checkDatabase(db *sql.DB) Check {
	start := time.Now()
	
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return Check{
			Status:  "fail",
			Message: fmt.Sprintf("Database unreachable: %v", err),
		}
	}

	// Check if we can execute a simple query
	var result int
	err := db.QueryRowContext(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		return Check{
			Status:  "fail",
			Message: fmt.Sprintf("Database query failed: %v", err),
		}
	}

	duration := time.Since(start).Milliseconds()
	message := "Connected"
	status := "pass"
	
	// Warn if response is slow
	if duration > 100 {
		status = "warn"
		message = fmt.Sprintf("Connected but slow (%dms)", duration)
	}

	return Check{
		Status:  status,
		Message: message,
		Time:    fmt.Sprintf("%dms", duration),
	}
}

// checkDiskSpace verifies available disk space
func checkDiskSpace() Check {
	// On Windows, use GetDiskFreeSpaceEx via syscall
	path, err := os.Getwd()
	if err != nil {
		return Check{
			Status:  "warn",
			Message: "Could not check disk space",
		}
	}

	// Convert to UTF16 for Windows API
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return Check{
			Status:  "warn",
			Message: "Invalid path for disk check",
		}
	}

	var freeBytesAvailable uint64
	var totalBytes uint64
	var totalFreeBytes uint64

	// Call Windows API
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getDiskFreeSpaceEx := kernel32.NewProc("GetDiskFreeSpaceExW")
	
	ret, _, _ := getDiskFreeSpaceEx.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(unsafe.Pointer(&freeBytesAvailable)),
		uintptr(unsafe.Pointer(&totalBytes)),
		uintptr(unsafe.Pointer(&totalFreeBytes)),
	)

	if ret == 0 {
		return Check{
			Status:  "warn",
			Message: "Could not retrieve disk space",
		}
	}

	// Calculate usage
	usedBytes := totalBytes - freeBytesAvailable
	usedPercent := float64(usedBytes) / float64(totalBytes) * 100
	availableGB := float64(freeBytesAvailable) / (1024 * 1024 * 1024)
	
	message := fmt.Sprintf("%.2f GB available (%.1f%% used)", availableGB, usedPercent)

	// Fail if less than 1GB or >95% used
	if availableGB < 1 || usedPercent > 95 {
		return Check{
			Status:  "fail",
			Message: message,
		}
	}

	// Warn if less than 5GB or >85% used
	if availableGB < 5 || usedPercent > 85 {
		return Check{
			Status:  "warn",
			Message: message,
		}
	}

	return Check{
		Status:  "pass",
		Message: message,
	}
}

// checkMemory verifies memory usage
func checkMemory() Check {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	allocMB := float64(m.Alloc) / (1024 * 1024)
	sysMB := float64(m.Sys) / (1024 * 1024)
	
	message := fmt.Sprintf("Alloc: %.2f MB, Sys: %.2f MB", allocMB, sysMB)

	// Warn if using more than 500MB
	if allocMB > 500 {
		return Check{
			Status:  "warn",
			Message: message,
		}
	}

	return Check{
		Status:  "pass",
		Message: message,
	}
}

// getVersion returns the application version
func getVersion() string {
	// You can read this from a VERSION file or build-time variable
	version := os.Getenv("APP_VERSION")
	if version == "" {
		version = "dev"
	}
	return version
}

// getUptime returns how long the server has been running
func getUptime() string {
	duration := time.Since(startTime)
	
	days := int(duration.Hours() / 24)
	hours := int(duration.Hours()) % 24
	minutes := int(duration.Minutes()) % 60
	seconds := int(duration.Seconds()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm %ds", days, hours, minutes, seconds)
	} else if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}
