package sensor

import (
    "fmt"
    "github.com/nalandras/data-forwarder/sensor/driver"
    "sort"
)

var drivers = make(map[string]driver.Driver)

// Register makes a database driver available by the provided name.
// If Register is called twice with the same name or if driver is nil,
// it panics.
func Register(name string, driver driver.Driver) {
    if driver == nil {
        panic("sensor: Register driver is nil")
    }
    if _, dup := drivers[name]; dup {
        panic("sensor: Register called twice for driver " + name)
    }
    drivers[name] = driver
}

func unregisterAllDrivers() {
    // For tests.
    drivers = make(map[string]driver.Driver)
}

func GetDriver(name string) (driver driver.Driver) {
    driver = drivers[name]
    if driver == nil {
        panic(fmt.Sprintf("sensor: Can not find %s from register list", name))
    }
    return
}

// Drivers returns a sorted list of the names of the registered drivers.
func Drivers() []string {
    var list []string
    for name := range drivers {
        list = append(list, name)
    }
    sort.Strings(list)
    return list
}
