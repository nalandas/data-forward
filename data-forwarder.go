package main

import (
    "encoding/json"
    "flag"
    "fmt"
    "github.com/nalandras/data-forwarder/sensor"
    . "github.com/nalandras/data-forwarder/util"
    _ "github.com/nalandras/df-sensor/database"
    "log"
    "os"
    "runtime/pprof"
    "time"
)

func EmitOptions() {
    Emit("\t--- options -------\n")
    Emit("\tconfig-arg:          %s\n", Options.ConfigArg)
    Emit("\tidle-timeout:        %v\n", Options.IdleTimeout)
    Emit("\tspool-size:          %d\n", Options.SpoolSize)
    Emit("\tharvester-buff-size: %d\n", Options.HarvesterBufferSize)
    Emit("\t--- flags ---------\n")
    Emit("\ttail (on-rotation):  %t\n", Options.TailOnRotate)
    Emit("\tlog-to-syslog:          %t\n", Options.UseSyslog)
    Emit("\tquiet:             %t\n", Options.Quiet)
    if runProfiler() {
        Emit("\t--- profile run ---\n")
        Emit("\tcpu-profile-file:    %s\n", Options.CpuProfileFile)
    }

}

// exits with stat existStat.usageError if required options are not provided
func assertRequiredOptions() {
    if Options.ConfigArg == "" {
        Exit(ExitStat.UsageError, "fatal: config file must be defined")
    }
}

const logflags = log.Ldate | log.Ltime | log.Lmicroseconds

func init() {
    flag.StringVar(&Options.ConfigArg, "config", Options.ConfigArg, "path to logstash-forwarder configuration file or directory")

    flag.StringVar(&Options.CpuProfileFile, "cpuprofile", Options.CpuProfileFile, "path to cpu profile output - note: exits on profile end.")

    flag.Uint64Var(&Options.SpoolSize, "spool-size", Options.SpoolSize, "event count spool threshold - forces network flush")
    flag.Uint64Var(&Options.SpoolSize, "sv", Options.SpoolSize, "event count spool threshold - forces network flush")

    flag.IntVar(&Options.HarvesterBufferSize, "harvest-buffer-size", Options.HarvesterBufferSize, "harvester reader buffer size")
    flag.IntVar(&Options.HarvesterBufferSize, "hb", Options.HarvesterBufferSize, "harvester reader buffer size")

    flag.BoolVar(&Options.UseSyslog, "log-to-syslog", Options.UseSyslog, "log to syslog instead of stdout") // deprecate this
    flag.BoolVar(&Options.UseSyslog, "syslog", Options.UseSyslog, "log to syslog instead of stdout")

    flag.BoolVar(&Options.TailOnRotate, "tail", Options.TailOnRotate, "always tail on log rotation -note: may skip entries ")
    flag.BoolVar(&Options.TailOnRotate, "t", Options.TailOnRotate, "always tail on log rotation -note: may skip entries ")

    flag.BoolVar(&Options.Quiet, "quiet", Options.Quiet, "operate in quiet mode - only Emit errors to log")
    flag.BoolVar(&Options.Version, "version", Options.Version, "output the version of this program")
}

func init() {
    log.SetFlags(logflags)
}

func main() {
    defer func() {
        p := recover()
        if p == nil {
            return
        }
        Fault("recovered panic: %v", p)
    }()

    flag.Parse()

    if Options.Version {
        fmt.Println(Version)
        return
    }

    if Options.UseSyslog {
        configureSyslog()
    }

    assertRequiredOptions()
    EmitOptions()

    if runProfiler() {
        f, err := os.Create(Options.CpuProfileFile)
        if err != nil {
            log.Fatal(err)
        }
        pprof.StartCPUProfile(f)
        Emit("Profiling enabled. I will collect profiling information and then exit in 60 seconds.")
        go func() {
            time.Sleep(60 * time.Second)
            pprof.StopCPUProfile()
            panic("60-seconds of profiling is complete. Shutting down.")
        }()
    }

    config_files, err := DiscoverConfigs(Options.ConfigArg)
    if err != nil {
        Fault("Could not use -config of '%s': %s", Options.ConfigArg, err)
    }

    var config Config

    for _, filename := range config_files {
        additional_config, err := LoadConfig(filename)
        if err == nil {
            var result Config
            err = DecodeStruct(&result, additional_config)
            if err == nil {
                err = MergeConfig(&config, result)
            }
        }
        if err != nil {
            Fault("Could not load config file %s: %s", filename, err)
        }
    }
    FinalizeConfig(&config)

    // Load sensors' configuration
    prepareSensor()

    event_chan := make(chan *DataEvent, 16)
    publisher_chan := make(chan []*DataEvent, 1)
    registrar_chan := make(chan []*DataEvent, 1)

    // The basic model of execution:
    // - prospector: finds files in paths/globs to harvest, starts harvesters
    // - harvester: reads a file, sends events to the spooler
    // - spooler: buffers events until ready to flush to the publisher
    // - publisher: writes to the network, notifies registrar
    // - registrar: records positions of files read
    // Finally, prospector uses the registrar information, on restart, to
    // determine where in each file to restart a harvester.

    restart := &ProspectorResume{}
    restart.persist = make(chan *DataState)

    // Load the previous log file locations now, for use in prospector
    restart.files = make(map[string]*DataState)
    if existing, e := os.Open(".data-forwarder"); e == nil {
        defer existing.Close()
        wd := ""
        if wd, e = os.Getwd(); e != nil {
            Emit("WARNING: os.Getwd returned unexpected error %s -- ignoring\n", e.Error())
        }
        Emit("Loading registrar data from %s/.data-forwarder\n", wd)

        decoder := json.NewDecoder(existing)
        decoder.Decode(&restart.files)
    }

    pendingProspectorCnt := 0

    // Prospect the globs/paths given on the command line and launch harvesters
    for _, fileconfig := range config.Files {
        prospector := &Prospector{FileConfig: fileconfig}
        go prospector.Prospect(restart, event_chan)
        pendingProspectorCnt++
    }

    // Now determine which states we need to persist by pulling the events from the prospectors
    // When we hit a nil source a prospector had finished so we decrease the expected events
    Emit("Waiting for %d prospectors to initialise\n", pendingProspectorCnt)
    persist := make(map[string]*FileState)

    for event := range restart.persist {
        if event.Source == nil {
            pendingProspectorCnt--
            if pendingProspectorCnt == 0 {
                break
            }
            continue
        }
        persist[*event.Source] = event
        Emit("Registrar will re-save state for %s\n", *event.Source)
    }

    Emit("All prospectors initialised with %d states to persist\n", len(persist))

    // Harvesters dump events into the spooler.
    go Spool(event_chan, publisher_chan, Options.SpoolSize, Options.IdleTimeout)

    go Publishv1(publisher_chan, registrar_chan, &config.Network)

    // registrar records last acknowledged positions in all files.
    Registrar(persist, registrar_chan)
}

func runProfiler() bool {
    return Options.CpuProfileFile != ""
}

func prepareSensor() {
    for _, driver := range sensor.Drivers() {
        sensor.GetDriver(driver).LoadConfig(Options.ConfigArg)
    }
}
