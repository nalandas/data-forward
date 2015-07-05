package main

import (
    "fmt"
    . "github.com/nalandras/data-forwarder/util"
    "time"
)

var defaultConfig = &struct {
    netTimeout   int64
    fileDeadtime string
}{
    netTimeout:   15,
    fileDeadtime: "24h",
}

type Config struct {
    Network NetworkConfig `json:network`
    Files   []FileConfig  `json:files`
}

type NetworkConfig struct {
    Servers        []string `json:servers`
    SSLCertificate string   `json:"ssl certificate"`
    SSLKey         string   `json:"ssl key"`
    SSLCA          string   `json:"ssl ca"`
    Timeout        int64    `json:timeout`
    timeout        time.Duration
}

type FileConfig struct {
    Paths    []string          `json:paths`
    Fields   map[string]string `json:fields`
    DeadTime string            `json:"dead time"`
    deadtime time.Duration
}

// Append values to the 'to' config from the 'from' config, erroring
// if a value would be overwritten by the merge.
func MergeConfig(to *Config, from Config) (err error) {

    to.Network.Servers = append(to.Network.Servers, from.Network.Servers...)
    to.Files = append(to.Files, from.Files...)

    // TODO: Is there a better way to do this in Go?
    if from.Network.SSLCertificate != "" {
        if to.Network.SSLCertificate != "" {
            return fmt.Errorf("SSLCertificate already defined as '%s' in previous config file", to.Network.SSLCertificate)
        }
        to.Network.SSLCertificate = from.Network.SSLCertificate
    }
    if from.Network.SSLKey != "" {
        if to.Network.SSLKey != "" {
            return fmt.Errorf("SSLKey already defined as '%s' in previous config file", to.Network.SSLKey)
        }
        to.Network.SSLKey = from.Network.SSLKey
    }
    if from.Network.SSLCA != "" {
        if to.Network.SSLCA != "" {
            return fmt.Errorf("SSLCA already defined as '%s' in previous config file", to.Network.SSLCA)
        }
        to.Network.SSLCA = from.Network.SSLCA
    }
    if from.Network.Timeout != 0 {
        if to.Network.Timeout != 0 {
            return fmt.Errorf("Timeout already defined as '%d' in previous config file", to.Network.Timeout)
        }
        to.Network.Timeout = from.Network.Timeout
    }
    return nil
}

func FinalizeConfig(config *Config) (err error) {
    if config.Network.Timeout == 0 {
        config.Network.Timeout = defaultConfig.netTimeout
    }

    config.Network.timeout = time.Duration(config.Network.Timeout) * time.Second

    for k, _ := range config.Files {
        if config.Files[k].DeadTime == "" {
            config.Files[k].DeadTime = defaultConfig.fileDeadtime
        }
        config.Files[k].deadtime, err = time.ParseDuration(config.Files[k].DeadTime)
        if err != nil {
            Emit("Failed to parse dead time duration '%s'. Error was: %s\n", config.Files[k].DeadTime, err)
            return
        }
    }
    return nil
}
