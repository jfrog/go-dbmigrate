# Generic Go Methods Driver

* Runs pre-registered Golang methods that receive no parameters and return `error` on failure.
* Stores migration version details in auto-generated table ``db_migrations`` of database that user should provide in driver initialization url.
  The url should be same as the database connection string, but the schema will be `gomethods`. 
  and the real schema (database type) (database type) should be provided in `migrations_db_type` query parameter (see example below)
  This parameter will be stripped of before the driver will attempt to connect to the database.
  Currently only `postgres` schema is supported.  
  

## Usage in Go

```go
import "github.com/jfrog/go-dbmigrate/migrate"

// Import your migration methods package so that they are registered and available for the GoMethods driver.
// There is no need to import the GoMethods driver explicitly, as it should already be imported by your migration methods package.
import _ "my_go_methods_migrator"

// use synchronous versions of migration functions ...
allErrors, ok := migrate.UpSync("gomethods://user@host:port/database?migrations_db_type=postgres", "./path")
if !ok {
  fmt.Println("Oh no ...")
  // do sth with allErrors slice
}

// use the asynchronous version of migration functions ...
pipe := migrate.NewPipe()
go migrate.Up(pipe, "gomethods://user@host:port/database?migrations_db_type=postgres", "./path")
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

001_first_release.up.gom
```
V001_some_migration_operation_up
V001_some_other_operation_up
...
```

001_first_release.down.gom
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

 