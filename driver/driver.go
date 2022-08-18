// Package driver holds the driver interface.
package driver

import (
	"fmt"
	neturl "net/url" // alias to allow `url string` func signature in New

	"github.com/jfrog/go-dbmigrate/file"
)

var (
	ErrLocked = fmt.Errorf("can't acquire lock")
)

// Driver is the interface type that needs to implemented by all drivers.
type Driver interface {

	// Initialize is the first function to be called.
	// Check the url string and open and verify any connection
	// that has to be made.
	Initialize(url string, initOptions ...func(Driver)) error

	// Close is the last function to be called.
	// Close any open connection here.
	Close() error

	// FilenameExtension returns the extension of the migration files.
	// The returned string must not begin with a dot.
	FilenameExtension() string

	// Migrate is the heart of the driver.
	// It will receive a file which the driver should apply
	// to its backend or whatever. The migration function should use
	// the pipe channel to return any errors or other useful information.
	Migrate(file file.File, pipe chan interface{})

	// Version returns the current migration version.
	Version() (uint64, error)
}

type DriverGenerator struct {
	fnGenerator   func() Driver
	fnInitOptions []func(Driver)
}

func NewDriverGenerator(fn func() Driver) *DriverGenerator {
	return &DriverGenerator{
		fnGenerator: fn,
	}
}

func (dg *DriverGenerator) RegisterInitFunction(fnInit func(Driver)) {
	dg.fnInitOptions = append(dg.fnInitOptions, fnInit)
}

func (dg *DriverGenerator) Generate() Driver {
	res := dg.fnGenerator()
	for _, option := range dg.fnInitOptions {
		option(res)
	}
	return res
}

// New returns Driver and calls Initialize on it
func New(url string, initOptions ...func(Driver)) (Driver, error) {
	u, err := neturl.Parse(url)
	if err != nil {
		return nil, err
	}

	gen, exists := GetDriverGenerator(u.Scheme)
	if !exists {
		return nil, fmt.Errorf("Driver '%s' not found.", u.Scheme)
	}
	d := gen.Generate()
	verifyFilenameExtension(u.Scheme, d)
	if err := d.Initialize(url, initOptions...); err != nil {
		return nil, err
	}

	return d, nil
}

// verifyFilenameExtension panics if the driver's filename extension
// is not correct or empty.
func verifyFilenameExtension(driverName string, d Driver) {
	f := d.FilenameExtension()
	if f == "" {
		panic(fmt.Sprintf("%s.FilenameExtension() returns empty string.", driverName))
	}
	if f[0:1] == "." {
		panic(fmt.Sprintf("%s.FilenameExtension() returned string must not start with a dot.", driverName))
	}
}
