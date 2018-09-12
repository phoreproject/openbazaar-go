// Copyright (c) 2013-2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package btclog

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
}
