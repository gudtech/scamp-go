package scamp

import (
	"log"
	"os"
)

var (
	// Trace is a nil logger that uses nullwriter
	Trace *log.Logger
	// Info wraps log.logger (os.Stdout) and formats log entries as `"INFO: ", log.Ldate|log.Ltime|log.Lshortfile`
	Info *log.Logger
	// Warning wraps log.logger (os.Stdout) and formats log entries as `"WARNING: ", log.Ldate|log.Ltime|log.Lshortfile`
	Warning *log.Logger
	// Error wraps log.logger (os.Stdout) and formats log entries as `"ERROR: ", log.Ldate|log.Ltime|log.Lshortfile`
	Error *log.Logger
)

type nullWriter int

func (nullWriter) Write([]byte) (int, error) { return 0, nil }

func initSCAMPLogger() {
	// Idempotent logger setup!
	if Trace != nil {
		return
	}

	Trace = log.New(new(nullWriter), "TRACE: ", log.Ldate|log.Ltime|log.Lshortfile)
	// Trace = log.New(os.Stdout, "TRACE: ", log.Ldate|log.Ltime|log.Lshortfile)
	Info = log.New(os.Stdout, "INFO:  ", log.Ldate|log.Ltime|log.Lshortfile)
	Warning = log.New(os.Stdout, "WARN:  ", log.Ldate|log.Ltime|log.Lshortfile)
	Error = log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}
