package logger

import (
	"io"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

// Color constants
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
	ColorBold   = "\033[1m"

	// Background colors
	BgRed    = "\033[41m"
	BgGreen  = "\033[42m"
	BgYellow = "\033[43m"
)

type Logger struct {
	*logrus.Logger
	isColorEnabled bool
}

func New() *Logger {
	log := logrus.New()

	// Set default format with color support
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
		ForceColors:     true,
		DisableColors:   false,
	})

	// Default to info level
	log.SetLevel(logrus.InfoLevel)

	// Default output to stdout
	log.SetOutput(os.Stdout)

	return &Logger{
		Logger:         log,
		isColorEnabled: true,
	}
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

		// For file output, disable colors and create dual output
		l.isColorEnabled = false
		l.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
			DisableColors:   true,
		})
		l.SetOutput(io.MultiWriter(os.Stdout, file))
	} else {
		// For stdout only, keep colors enabled
		l.isColorEnabled = true
		l.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
			ForceColors:     true,
		})
	}

	return nil
}

// Success logs with prominent green styling
func (l *Logger) Success(msg string) {
	if l.isColorEnabled {
		l.WithField("status", "success").Infof("%s%s🎉 SUCCESS%s %s", BgGreen, ColorBold, ColorReset, msg)
	} else {
		l.WithField("status", "success").Infof("✅ SUCCESS: %s", msg)
	}
}

// Successf logs with prominent green styling and formatting
func (l *Logger) Successf(format string, args ...interface{}) {
	if l.isColorEnabled {
		l.WithField("status", "success").Infof("%s%s🎉 SUCCESS%s "+format, append([]interface{}{BgGreen, ColorBold, ColorReset}, args...)...)
	} else {
		l.WithField("status", "success").Infof("✅ SUCCESS: "+format, args...)
	}
}

// ErrorHighlight logs error with prominent red styling
func (l *Logger) ErrorHighlight(msg string) {
	if l.isColorEnabled {
		l.WithField("status", "error").Errorf("%s%s❌ ERROR%s %s", BgRed, ColorBold, ColorReset, msg)
	} else {
		l.WithField("status", "error").Errorf("❌ ERROR: %s", msg)
	}
}

// ErrorHighlightf logs error with prominent red styling and formatting
func (l *Logger) ErrorHighlightf(format string, args ...interface{}) {
	if l.isColorEnabled {
		l.WithField("status", "error").Errorf("%s%s❌ ERROR%s "+format, append([]interface{}{BgRed, ColorBold, ColorReset}, args...)...)
	} else {
		l.WithField("status", "error").Errorf("❌ ERROR: "+format, args...)
	}
}

// WarnHighlight logs warning with prominent yellow styling
func (l *Logger) WarnHighlight(msg string) {
	if l.isColorEnabled {
		l.WithField("status", "warning").Warnf("%s%s⚠️ WARNING%s %s", BgYellow, ColorBold, ColorReset, msg)
	} else {
		l.WithField("status", "warning").Warnf("⚠️ WARNING: %s", msg)
	}
}

// WarnHighlightf logs warning with prominent yellow styling and formatting
func (l *Logger) WarnHighlightf(format string, args ...interface{}) {
	if l.isColorEnabled {
		l.WithField("status", "warning").Warnf("%s%s⚠️ WARNING%s "+format, append([]interface{}{BgYellow, ColorBold, ColorReset}, args...)...)
	} else {
		l.WithField("status", "warning").Warnf("⚠️ WARNING: "+format, args...)
	}
}