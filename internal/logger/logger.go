package logger

import (
	"fmt"
	"io"
	"log"
	"os"
)

type LogLevel int

const (
	ERROR LogLevel = iota
	WARN
	INFO
	DEBUG
)

var (
	logLevel LogLevel = INFO
	logger   *log.Logger
)

func init() {
	logger = log.New(os.Stderr, "", 0)
}

func SetLevel(level LogLevel) {
	logLevel = level
}

func SetOutput(w io.Writer) {
	logger.SetOutput(w)
}

func SetVerbose(verbose bool) {
	if verbose {
		SetLevel(DEBUG)
	} else {
		SetLevel(INFO)
	}
}

func Error(format string, args ...interface{}) {
	if logLevel >= ERROR {
		msg := fmt.Sprintf("❌ ERROR: "+format, args...)
		logger.Println(msg)
	}
}

func Warn(format string, args ...interface{}) {
	if logLevel >= WARN {
		msg := fmt.Sprintf("⚠️  WARN: "+format, args...)
		logger.Println(msg)
	}
}

func Info(format string, args ...interface{}) {
	if logLevel >= INFO {
		msg := fmt.Sprintf("ℹ️  INFO: "+format, args...)
		logger.Println(msg)
	}
}

func Debug(format string, args ...interface{}) {
	if logLevel >= DEBUG {
		msg := fmt.Sprintf("🐛 DEBUG: "+format, args...)
		logger.Println(msg)
	}
}

func Print(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}

func Println(args ...interface{}) {
	fmt.Println(args...)
}
