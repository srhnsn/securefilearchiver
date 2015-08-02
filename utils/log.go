package utils

import (
	"io"
	"io/ioutil"
	"log"
	"os"
)

const (
	logParams = log.Ldate | log.Ltime | log.Lshortfile
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

	// CurrentLoggerConfig holds the currently in use LoggerConfig.
	CurrentLoggerConfig LoggerConfig
	// DefaultLoggerConfig is a LoggerConfig where some logging levels are printed
	// to the console (errors to stderr, others to stdout).
	DefaultLoggerConfig = LoggerConfig{
		Trace:   []io.Writer{ioutil.Discard},
		Info:    []io.Writer{os.Stdout},
		Warning: []io.Writer{os.Stdout},
		Error:   []io.Writer{os.Stderr},
	}
)

// LoggerConfig holds the output interfaces for the different logging levels.
type LoggerConfig struct {
	Trace   []io.Writer
	Info    []io.Writer
	Warning []io.Writer
	Error   []io.Writer
}

func init() {
	applyLoggerConfig(DefaultLoggerConfig)
}

// InitLogger initializes the logging configuration.
func InitLogger(logFilename string, quiet bool, verbose bool) {
	config := DefaultLoggerConfig

	if verbose {
		config.Trace = []io.Writer{os.Stdout}
	}

	if quiet {
		config.Trace = []io.Writer{ioutil.Discard}
		config.Info = []io.Writer{ioutil.Discard}
		config.Warning = []io.Writer{ioutil.Discard}
	}

	if len(logFilename) != 0 {
		logFile, err := os.OpenFile(logFilename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)

		if err != nil {
			panic(err)
		}

		config.Trace = append(config.Trace, logFile)
		config.Info = append(config.Info, logFile)
		config.Warning = append(config.Warning, logFile)
		config.Error = append(config.Error, logFile)
	}

	applyLoggerConfig(config)
}

func applyLoggerConfig(config LoggerConfig) {
	CurrentLoggerConfig = config

	if len(config.Trace) == 0 {
		config.Trace = []io.Writer{ioutil.Discard}
	}

	if len(config.Info) == 0 {
		config.Info = []io.Writer{ioutil.Discard}
	}

	if len(config.Warning) == 0 {
		config.Warning = []io.Writer{ioutil.Discard}
	}

	if len(config.Error) == 0 {
		config.Error = []io.Writer{ioutil.Discard}
	}

	Trace = log.New(io.MultiWriter(config.Trace...), "TRACE:   ", logParams)
	Info = log.New(io.MultiWriter(config.Info...), "INFO:    ", logParams)
	Warning = log.New(io.MultiWriter(config.Warning...), "WARNING: ", logParams)
	Error = log.New(io.MultiWriter(config.Error...), "ERROR:   ", logParams)
}
