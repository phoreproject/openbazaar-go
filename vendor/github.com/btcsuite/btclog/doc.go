<<<<<<< HEAD
// Copyright (c) 2013-2014 The btcsuite developers
=======
// Copyright (c) 2013-2017 The btcsuite developers
>>>>>>> 159c03b9... IPFS rebase second pass
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

/*
<<<<<<< HEAD
Package btclog implements a subsystem aware logger backed by seelog.

Seelog allows you to specify different levels per backend such as console and
file, but it doesn't support levels per subsystem well.  You can create multiple
loggers, but when those are backed by a file, they have to go to different
files.  That is where this package comes in.  It provides a SubsystemLogger
which accepts the backend seelog logger to do the real work.  Each instance of a
SubsystemLogger then allows you specify (and retrieve) an individual level per
subsystem.  All messages are then passed along to the backend seelog logger.
=======
Package btclog defines an interface and default implementation for subsystem
logging.

Log level verbosity may be modified at runtime for each individual subsystem
logger.

The default implementation in this package must be created by the Backend type.
Backends can write to any io.Writer, including multi-writers created by
io.MultiWriter.  Multi-writers allow log output to be written to many writers,
including standard output and log files.

Optional logging behavior can be specified by using the LOGFLAGS environment
variable and overridden per-Backend by using the WithFlags call option. Multiple
LOGFLAGS options can be specified, separated by commas.  The following options
are recognized:

  longfile: Include the full filepath and line number in all log messages

  shortfile: Include the filename and line number in all log messages.
  Overrides longfile.
>>>>>>> 159c03b9... IPFS rebase second pass
*/
package btclog
