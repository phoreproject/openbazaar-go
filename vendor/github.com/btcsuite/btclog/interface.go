<<<<<<< HEAD
// Copyright (c) 2013-2014 The btcsuite developers
=======
// Copyright (c) 2013-2017 The btcsuite developers
>>>>>>> 159c03b9... IPFS rebase second pass
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package btclog

<<<<<<< HEAD
import (
	"fmt"
)

// LogLevel is the level at which a logger is configured.  All messages sent
// to a level which is below the current level are filtered.
type LogLevel uint8

// LogLevel contants.
const (
	TraceLvl LogLevel = iota
	DebugLvl
	InfoLvl
	WarnLvl
	ErrorLvl
	CriticalLvl
	Off
)

// Map of log levels to string representations.
var logLevelStrings = map[LogLevel]string{
	TraceLvl:    "trace",
	DebugLvl:    "debug",
	InfoLvl:     "info",
	WarnLvl:     "warn",
	ErrorLvl:    "error",
	CriticalLvl: "critical",
	Off:         "off",
}

// String converts level to a human-readable string.
func (level LogLevel) String() string {
	if s, ok := logLevelStrings[level]; ok {
		return s
	}

	return fmt.Sprintf("Unknown LogLevel (%d)", uint8(level))
}

// Logger is an interface which describes a level-based logger.
type Logger interface {
	// Tracef formats message according to format specifier and writes to
	// to log with TraceLvl.
	Tracef(format string, params ...interface{})

	// Debugf formats message according to format specifier and writes to
	// log with DebugLvl.
	Debugf(format string, params ...interface{})

	// Infof formats message according to format specifier and writes to
	// log with InfoLvl.
	Infof(format string, params ...interface{})

	// Warnf formats message according to format specifier and writes to
	// to log with WarnLvl.
	Warnf(format string, params ...interface{}) error

	// Errorf formats message according to format specifier and writes to
	// to log with ErrorLvl.
	Errorf(format string, params ...interface{}) error

	// Criticalf formats message according to format specifier and writes to
	// log with CriticalLvl.
	Criticalf(format string, params ...interface{}) error

	// Trace formats message using the default formats for its operands
	// and writes to log with TraceLvl.
	Trace(v ...interface{})

	// Debug formats message using the default formats for its operands
	// and writes to log with DebugLvl.
	Debug(v ...interface{})

	// Info formats message using the default formats for its operands
	// and writes to log with InfoLvl.
	Info(v ...interface{})

	// Warn formats message using the default formats for its operands
	// and writes to log with WarnLvl.
	Warn(v ...interface{}) error

	// Error formats message using the default formats for its operands
	// and writes to log with ErrorLvl.
	Error(v ...interface{}) error

	// Critical formats message using the default formats for its operands
	// and writes to log with CriticalLvl.
	Critical(v ...interface{}) error

	// Level returns the current logging level.
	Level() LogLevel

	// SetLevel changes the logging level to the passed level.
	SetLevel(level LogLevel)

	// Close the logger.  Any future log messages will be ignored.
	Close()
=======
// Logger is an interface which describes a level-based logger.  A default
// implementation of Logger is implemented by this package and can be created
// by calling (*Backend).Logger.
type Logger interface {
	// Tracef formats message according to format specifier and writes to
	// to log with LevelTrace.
	Tracef(format string, params ...interface{})

	// Debugf formats message according to format specifier and writes to
	// log with LevelDebug.
	Debugf(format string, params ...interface{})

	// Infof formats message according to format specifier and writes to
	// log with LevelInfo.
	Infof(format string, params ...interface{})

	// Warnf formats message according to format specifier and writes to
	// to log with LevelWarn.
	Warnf(format string, params ...interface{})

	// Errorf formats message according to format specifier and writes to
	// to log with LevelError.
	Errorf(format string, params ...interface{})

	// Criticalf formats message according to format specifier and writes to
	// log with LevelCritical.
	Criticalf(format string, params ...interface{})

	// Trace formats message using the default formats for its operands
	// and writes to log with LevelTrace.
	Trace(v ...interface{})

	// Debug formats message using the default formats for its operands
	// and writes to log with LevelDebug.
	Debug(v ...interface{})

	// Info formats message using the default formats for its operands
	// and writes to log with LevelInfo.
	Info(v ...interface{})

	// Warn formats message using the default formats for its operands
	// and writes to log with LevelWarn.
	Warn(v ...interface{})

	// Error formats message using the default formats for its operands
	// and writes to log with LevelError.
	Error(v ...interface{})

	// Critical formats message using the default formats for its operands
	// and writes to log with LevelCritical.
	Critical(v ...interface{})

	// Level returns the current logging level.
	Level() Level

	// SetLevel changes the logging level to the passed level.
	SetLevel(level Level)
>>>>>>> 159c03b9... IPFS rebase second pass
}
