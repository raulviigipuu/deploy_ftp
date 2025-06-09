package log

import (
	"io"
	"log"
	"os"
)

var (
	infoLogger  *log.Logger
	errorLogger *log.Logger
)

// Init initializes the log system with optional output destination.
// If 'out' is nil, logs go to os.Stdout (info) and os.Stderr (error).
func Init(out io.Writer) {
	if out == nil {
		infoLogger = log.New(os.Stdout, "INFO  ", log.LstdFlags)
		errorLogger = log.New(os.Stderr, "ERROR ", log.LstdFlags)
	} else {
		infoLogger = log.New(out, "INFO  ", log.LstdFlags)
		errorLogger = log.New(out, "ERROR ", log.LstdFlags)
	}
}

// Info logs a plain info message.
func Info(msg string) {
	infoLogger.Println(msg)
}

// Error logs a plain error message.
func Error(msg string) {
	errorLogger.Println(msg)
}

// ErrorErr logs an error if it's not nil.
func ErrorErr(err error) {
	if err != nil {
		errorLogger.Println(err)
	}
}

// Fatal logs an error and exits the program with code 1.
func Fatal(msg string) {
	errorLogger.Fatal(msg) // logs and calls os.Exit(1)
}

// Takes an error if it's not nil.
func FatalErr(err error) {
	if err != nil {
		errorLogger.Fatal(err)
	}
}
