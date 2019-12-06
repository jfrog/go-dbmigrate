package driver

import (
	"sort"
	"sync"
)

var driversMu sync.Mutex
var drivers = make(map[string]*DriverGenerator)

// Registers a driver so it can be created from its name. Drivers should
// call this from an init() function so that they registers themselvse on
// import
func RegisterDriver(name string, driver *DriverGenerator) {
	driversMu.Lock()
	defer driversMu.Unlock()
	if driver.fnGenerator == nil {
		panic("driver: Register driver is nil")
	}
	if _, dup := drivers[name]; dup {
		panic("sql: Register called twice for driver " + name)
	}
	drivers[name] = driver
}

// Retrieves a registered driver by name
func GetDriverGenerator(name string) (*DriverGenerator, bool) {
	driversMu.Lock()
	defer driversMu.Unlock()
	driver, ok := drivers[name]
	return driver, ok
}

// Drivers returns a sorted list of the names of the registered drivers.
func Drivers() []string {
	driversMu.Lock()
	defer driversMu.Unlock()
	var list []string
	for name := range drivers {
		list = append(list, name)
	}
	sort.Strings(list)
	return list
}
