package utils

import (
	"io"
	"io/ioutil"
	"log"
	"os"
)

var (
	// Trace is a logging interface for trace-level information.
	Trace *log.Logger
	// Info is a logging interface for info-level information.
	Info *log.Logger
	// Warning is a logging interface for warning-level information.
	Warning *log.Logger
	// Error is a logging interface for error-level information.
	Error *log.Logger
)

// LoggerConfig holds the output interfaces for the different logging levels.
type LoggerConfig struct {
	Trace   io.Writer
	Info    io.Writer
	Warning io.Writer
	Error   io.Writer
}

// CurrentLoggerConfig holds the currently in use LoggerConfig.
var CurrentLoggerConfig LoggerConfig

// DefaultLoggerConfig is a LoggerConfig where all logging levels are printed
// to the console (errors to stderr, others to stdout).
var DefaultLoggerConfig = LoggerConfig{
	Trace:   os.Stdout,
	Info:    os.Stdout,
	Warning: os.Stdout,
	Error:   os.Stderr,
}

const logParams = log.Ldate | log.Ltime | log.Lshortfile

func init() {
	traceHandle := os.Stdout
	infoHandle := os.Stdout
	warningHandle := os.Stdout
	errorHandle := os.Stderr

	Trace = log.New(traceHandle, "TRACE:   ", logParams)
	Info = log.New(infoHandle, "INFO:    ", logParams)
	Warning = log.New(warningHandle, "WARNING: ", logParams)
	Error = log.New(errorHandle, "ERROR:   ", logParams)
}

// SetQuietLogging mutes all logging levels except for errors.
func SetQuietLogging() {
	config := DefaultLoggerConfig
	config.Trace = ioutil.Discard
	config.Info = ioutil.Discard
	config.Warning = ioutil.Discard

	applyLoggerConfig(config)
}

// SetVerboseLogging enables trace-level logging.
func SetVerboseLogging(verbose bool) {
	var handle io.Writer

	if verbose {
		handle = os.Stdout
	} else {
		handle = ioutil.Discard
	}

	config := DefaultLoggerConfig
	config.Trace = handle

	applyLoggerConfig(config)
}

func applyLoggerConfig(config LoggerConfig) {
	CurrentLoggerConfig = config

	Trace = log.New(config.Trace, "TRACE:   ", logParams)
	Info = log.New(config.Info, "INFO:    ", logParams)
	Warning = log.New(config.Warning, "WARNING: ", logParams)
	Error = log.New(config.Error, "ERROR:   ", logParams)
}
