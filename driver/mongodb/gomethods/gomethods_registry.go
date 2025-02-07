package gomethods

import (
	"fmt"
	"github.com/jfrog/go-dbmigrate/driver"
	"sync"
)

var methodsReceiversMu sync.Mutex

// Registers a methods receiver for go methods driver
// Users of gomethods migration drivers should call this method
// to register objects with their migration methods before executing the migration
func RegisterMethodsReceiverForDriver(driverName string, receiver interface{}) {
	methodsReceiversMu.Lock()
	defer methodsReceiversMu.Unlock()
	if receiver == nil {
		panic("Go methods: Register receiver object is nil")
	}

	gen, exists := driver.GetDriverGenerator(driverName)
	if !exists {
		panic("Go methods: Trying to register receiver for not registered driver " + driverName)
	}

	driverObj := gen.Generate()
	_, ok := driverObj.(GoMethodsDriver)
	if !ok {
		panic("Go methods: Trying to register receiver for non go methods driver " + driverName)
	}

	gen.RegisterInitFunction(func(d driver.Driver) {
		methodsDriver, ok := d.(GoMethodsDriver)
		if !ok {
			panic("Go methods: Trying to register receiver for non go methods driver " + driverName)
		}
		if methodsDriver.MethodsReceiver() != nil {
			panic("Go methods: Methods receiver already registered for driver " + driverName)
		}

		if err := methodsDriver.SetMethodsReceiver(receiver); err != nil {
			panic(fmt.Sprintf("Go methods: Failed to set methods receiver for driver %s\nError: %v",
				driverName, err))
		}
	})
}
