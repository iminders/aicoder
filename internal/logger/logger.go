package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

var (
	fileLogger *log.Logger
	verbose    bool
)

// Init initialises the logger, writing to ~/.aicoder/logs/.
func Init(verboseMode bool) {
	verbose = verboseMode
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	logDir := filepath.Join(home, ".aicoder", "logs")
	_ = os.MkdirAll(logDir, 0700)
	logFile := filepath.Join(logDir, fmt.Sprintf("aicoder-%s.log", time.Now().Format("20060102")))
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	fileLogger = log.New(f, "", log.LstdFlags|log.Lmicroseconds)
}

func Info(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if fileLogger != nil {
		fileLogger.Printf("[INFO]  " + msg)
	}
}

func Debug(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if fileLogger != nil {
		fileLogger.Printf("[DEBUG] " + msg)
	}
	if verbose {
		fmt.Fprintf(os.Stderr, "\033[90m[debug] "+msg+"\033[0m\n")
	}
}

func Error(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if fileLogger != nil {
		fileLogger.Printf("[ERROR] " + msg)
	}
	fmt.Fprintf(os.Stderr, "\033[31m[error] "+msg+"\033[0m\n")
}

func Warn(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if fileLogger != nil {
		fileLogger.Printf("[WARN]  " + msg)
	}
}
