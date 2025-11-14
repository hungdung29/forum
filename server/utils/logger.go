package utils

import (
	"fmt"
	"log"
	"os"
	"time"
)

// Logger provides structured logging
type Logger struct {
	logger *log.Logger
}

// NewLogger creates a new logger instance
func NewLogger() *Logger {
	return &Logger{
		logger: log.New(os.Stdout, "", log.LstdFlags),
	}
}

// Info logs informational messages
func (l *Logger) Info(msg string, fields ...interface{}) {
	l.log("INFO", msg, fields...)
}

// Error logs error messages
func (l *Logger) Error(msg string, fields ...interface{}) {
	l.log("ERROR", msg, fields...)
}

// Warn logs warning messages
func (l *Logger) Warn(msg string, fields ...interface{}) {
	l.log("WARN", msg, fields...)
}

// Debug logs debug messages
func (l *Logger) Debug(msg string, fields ...interface{}) {
	l.log("DEBUG", msg, fields...)
}

// log formats and outputs the log message with structured fields
func (l *Logger) log(level, msg string, fields ...interface{}) {
	timestamp := time.Now().Format("2006/01/02 15:04:05")
	output := fmt.Sprintf("[%s] %s: %s", level, timestamp, msg)
	
	// Add structured fields in key=value format
	if len(fields) > 0 {
		for i := 0; i < len(fields); i += 2 {
			if i+1 < len(fields) {
				output += fmt.Sprintf(" %v=%v", fields[i], fields[i+1])
			}
		}
	}
	
	l.logger.Println(output)
}

// HTTPLog logs HTTP request/response information
func (l *Logger) HTTPLog(method, path, ip string, statusCode int, duration time.Duration) {
	l.Info("HTTP Request",
		"method", method,
		"path", path,
		"ip", ip,
		"status", statusCode,
		"duration", duration.String(),
	)
}
