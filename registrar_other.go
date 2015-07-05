// +build !windows

package main

import (
    . "github.com/nalandras/data-forwarder/util"
    "os"
)

func onRegistryWrite(path, tempfile string) error {
    if e := os.Rename(tempfile, path); e != nil {
        Emit("registry rotate: rename of %s to %s - %s\n", tempfile, path, e)
        return e
    }
    return nil
}
