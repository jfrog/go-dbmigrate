# GoMethods Driver

* Runs pre-registered Golang methods that receive no parameters and return `error` on failure.
* Stores migration version details in table ``db_migrations`` of postgres database that user should provide in driver initialization.
  The ``db_migrations`` table will be auto-generated.
  

## Usage in Go

```go
import "github.com/jfrog/go-dbmigrate/migrate"

// Import your migration methods package so that they are registered and available for the GoMethods driver.
// There is no need to import the GoMethods driver explicitly, as it should already be imported by your migration methods package.
import _ "my_go_methods_migrator"

// use synchronous versions of migration functions ...
allErrors, ok := migrate.UpSync("postgres://user@host:port/database", "./path")
if !ok {
  fmt.Println("Oh no ...")
  // do sth with allErrors slice
}

// use the asynchronous version of migration functions ...
pipe := migrate.NewPipe()
go migrate.Up(pipe, "postgres://user@host:port/database", "./path")
// pipe is basically just a channel
// write your own channel listener. see writePipe() in main.go as an example.
```

## Migration files format

The migration files should have an ".gom" extension and contain a list of registered methods names.

Migration methods should satisfy the following:
* They should be exported (their name should start with a capital letter) 
* Their type should be `func () error`

Recommended (but not required) naming conventions for migration methods:
* Prefix with V<version> : for example V001 for version 1. 
* Suffix with "_up" or "_down" for up and down migrations correspondingly.

001_first_release.up.mgo
```
V001_some_migration_operation_up
V001_some_other_operation_up
...
```

001_first_release.down.mgo
```
V001_some_other_operation_down
V001_some_migration_operation_down
...
```

## Methods registration

For a detailed example see: [sample_migrator.go](https://github.com/jfrog/go-dbmigrate/blob/gomethods/driver/gomethods/example/sample_migrator.go)

```go
package my_go_methods_migrator

import (
  _ "github.com/jfrog/go-dbmigrate/driver/gomethods"
  "github.com/jfrog/go-dbmigrate/driver/mongodb/gomethods"
)

// common boilerplate
type MyGoMethodsMigrator struct {
}

func init() {
	gomethods.RegisterMethodsReceiverForDriver("gomethods", &MyGoMethodsMigrator{})
}


// Here goes the application-specific migration logic
func (r *MyGoMethodsMigrator) V001_some_migration_operation_up() error {
  // do something
  return nil
}

func (r *MyGoMethodsMigrator) V001_some_migration_operation_down() error {
  // revert some_migration_operation_up from above
  return nil
}

```

## Authors

* Demitry Gershovich, https://github.com/dimag-jfrog

 