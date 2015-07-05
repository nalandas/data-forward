package util

import (
    "log"
    "os"
    "time"
)

var ExitStat = struct {
    Ok, UsageError, Faulted int
}{
    Ok:         0,
    UsageError: 1,
    Faulted:    2,
}

var Options = &struct {
    ConfigArg           string
    SpoolSize           uint64
    HarvesterBufferSize int
    CpuProfileFile      string
    IdleTimeout         time.Duration
    UseSyslog           bool
    TailOnRotate        bool
    Quiet               bool
    Version             bool
}{
    SpoolSize:           1024,
    HarvesterBufferSize: 16 << 10,
    IdleTimeout:         time.Second * 5,
}

// REVU: yes, this is a temp hack.
func Emit(msgfmt string, args ...interface{}) {
    if Options.Quiet {
        return
    }
    log.Printf(msgfmt, args...)
}

func Fault(msgfmt string, args ...interface{}) {
    Exit(ExitStat.Faulted, msgfmt, args...)
}

func Exit(stat int, msgfmt string, args ...interface{}) {
    log.Printf(msgfmt, args...)
    os.Exit(stat)
}
