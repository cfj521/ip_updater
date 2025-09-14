package logger

import (
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

type Logger struct {
	*logrus.Logger
}

func New() *Logger {
	log := logrus.New()

	// Set default format
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	// Default to info level
	log.SetLevel(logrus.InfoLevel)

	// Default output to stdout
	log.SetOutput(os.Stdout)

	return &Logger{Logger: log}
}

func (l *Logger) Configure(level, filePath string, maxSize, maxAge int) error {
	// Set log level
	switch level {
	case "debug":
		l.SetLevel(logrus.DebugLevel)
	case "info":
		l.SetLevel(logrus.InfoLevel)
	case "warn":
		l.SetLevel(logrus.WarnLevel)
	case "error":
		l.SetLevel(logrus.ErrorLevel)
	default:
		l.SetLevel(logrus.InfoLevel)
	}

	// Create log file if specified
	if filePath != "" {
		// Create directory if it doesn't exist
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return err
		}

		file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return err
		}

		l.SetOutput(file)
	}

	return nil
}