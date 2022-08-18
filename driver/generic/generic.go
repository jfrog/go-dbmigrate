package generic

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/jfrog/go-dbmigrate/driver"
	"github.com/jfrog/go-dbmigrate/driver/mongodb/gomethods"
	"github.com/jfrog/go-dbmigrate/file"
	"github.com/jfrog/go-dbmigrate/migrate/direction"
	neturl "net/url" // alias to allow `url string` func signature in New
	"reflect"
)

type UnregisteredMethodsReceiverError string

func (e UnregisteredMethodsReceiverError) Error() string {
	return "Unregistered methods receiver for driver: " + string(e)
}

type WrongMethodsReceiverTypeError string

func (e WrongMethodsReceiverTypeError) Error() string {
	return "Wrong methods receiver type for driver: " + string(e)
}

const tableName = "db_migrations"
const DRIVER_NAME = "generic"

type Driver struct {
	db              *sql.DB
	methodsReceiver MethodsReceiver
	migrator        gomethods.Migrator
	url             string
	isLocked        bool
}

var _ gomethods.GoMethodsDriver = (*Driver)(nil)

type MethodsReceiver interface {
}

func (d *Driver) MethodsReceiver() interface{} {
	return d.methodsReceiver
}

func (d *Driver) SetMethodsReceiver(r interface{}) error {
	d.methodsReceiver = r
	return nil
}

func init() {
	driver.RegisterDriver("generic", driver.NewDriverGenerator(
		func() driver.Driver { return &Driver{} }))
}

func (driver *Driver) Initialize(url string, initOptions ...func(driver.Driver)) error {
	if driver.methodsReceiver == nil {
		return UnregisteredMethodsReceiverError(DRIVER_NAME)
	}
	urlObj, err := neturl.Parse(url)
	if err != nil {
		return fmt.Errorf("Failed to parse initialization url %s: %v", url, err)
	}
	queryValues := urlObj.Query()
	migrationsDb := queryValues.Get("migrations_db_type")
	var schema, driverName string
	switch migrationsDb {
	case "":
		return errors.New("db_migrations_database query parameter was not provider")
	case "postgres":
		schema = "postgres"
		driverName = "pgx"
	}
	if schema == "" {
		return fmt.Errorf("Could not deduce db migration database schema from url %s", url)
	}
	queryValues.Del("migrations_db_type")
	urlObj.RawQuery = queryValues.Encode()
	urlObj.Scheme = schema

	newUrl := urlObj.String()
	db, err := sql.Open(driverName, newUrl)
	if err != nil {
		return err
	}
	if err := db.Ping(); err != nil {
		return err
	}
	driver.db = db
	driver.url = newUrl

	if err := driver.ensureVersionTableExists(); err != nil {
		return err
	}

	driver.migrator = gomethods.Migrator{MethodInvoker: driver}
	return nil
}

func (driver *Driver) ensureConnectionNotClosed() error {
	pingErr := driver.db.Ping()
	if pingErr == nil {
		return nil
	}
	if pingErr.Error() != "sql: database is closed" {
		return pingErr
	}

	db, err := sql.Open("pgx", driver.url)
	if err != nil {
		return err
	}
	if err := db.Ping(); err != nil {
		return err
	}
	driver.db = db
	return nil
}

func (p *Driver) Close() error {
	if err := p.db.Close(); !driver.CanIgnoreError(err) {
		return err
	}
	return nil
}

// https://www.postgresql.org/docs/9.6/static/explicit-locking.html#ADVISORY-LOCKS
func (p *Driver) Lock() error {
	if p.isLocked {
		return driver.ErrLocked
	}

	aid, err := driver.GenerateAdvisoryLockId("xraydb", "migrate-generic")
	if err != nil {
		return err
	}

	// This will wait indefinitely until the lock can be acquired.
	query := `SELECT pg_advisory_lock($1)`
	if _, err := p.db.ExecContext(context.Background(), query, aid); err != nil {
		return fmt.Errorf("Generic try lock failed: %v", err)
	}

	p.isLocked = true
	return nil
}

func (p *Driver) Unlock() error {
	if !p.isLocked {
		return nil
	}

	aid, err := driver.GenerateAdvisoryLockId("xraydb", "migrate-generic")
	if err != nil {
		return err
	}

	query := `SELECT pg_advisory_unlock($1)`
	if _, err := p.db.ExecContext(context.Background(), query, aid); err != nil {
		return fmt.Errorf("Generic try unlock failed: %v", err)
	}
	p.isLocked = false
	return nil
}

func (driver *Driver) ensureVersionTableExists() (err error) {
	if err := driver.Lock(); err != nil {
		return err
	}

	defer func() {
		if e := driver.Unlock(); e != nil {
			if err == nil {
				err = e
			} else {
				err = fmt.Errorf("Error1: %v, Error2: %v", err, e)
			}
		}
	}()

	if _, err := driver.db.Exec("CREATE TABLE IF NOT EXISTS " + tableName + " (version int not null primary key);"); err != nil {
		return err
	}
	return nil
}

func (driver *Driver) FilenameExtension() string {
	return "gom"
}

func (driver *Driver) Version() (uint64, error) {
	if err := driver.ensureConnectionNotClosed(); err != nil {
		return 0, fmt.Errorf("failed to ensure db connection is open: %v", err)
	}

	var version uint64
	err := driver.db.QueryRow("SELECT version FROM " + tableName + " ORDER BY version DESC LIMIT 1").Scan(&version)
	switch {
	case err == sql.ErrNoRows:
		return 0, nil
	case err != nil:
		return 0, err
	default:
		return version, nil
	}
}

func (driver *Driver) Migrate(f file.File, pipe chan interface{}) {
	defer close(pipe)
	pipe <- f

	err := driver.migrator.Migrate(f, pipe)
	if err != nil {
		return
	}

	if err := driver.ensureConnectionNotClosed(); err != nil {
		pipe <- fmt.Errorf("failed to ensure db connection is open: %v", err)
		return
	}

	if f.Direction == direction.Up {
		if _, err := driver.db.Exec("INSERT INTO "+tableName+" (version) VALUES ($1)", f.Version); err != nil {
			pipe <- err
			return
		}
	} else if f.Direction == direction.Down {
		if _, err := driver.db.Exec("DELETE FROM "+tableName+" WHERE version=$1", f.Version); err != nil {
			pipe <- err
			return
		}
	}
}

func (driver *Driver) Validate(methodName string) error {
	methodWithReceiver, ok := reflect.TypeOf(driver.methodsReceiver).MethodByName(methodName)
	if !ok {
		return gomethods.MissingMethodError(methodName)
	}
	if methodWithReceiver.PkgPath != "" {
		return gomethods.MethodNotExportedError(methodName)
	}

	methodFunc := reflect.ValueOf(driver.methodsReceiver).MethodByName(methodName)
	methodTemplate := func() error { return nil }

	if methodFunc.Type() != reflect.TypeOf(methodTemplate) {
		return gomethods.WrongMethodSignatureError(methodName)
	}

	return nil
}

func (driver *Driver) Invoke(methodName string) error {
	name := methodName
	migrateMethod := reflect.ValueOf(driver.methodsReceiver).MethodByName(name)
	if !migrateMethod.IsValid() {
		return gomethods.MissingMethodError(methodName)
	}
	retValues := migrateMethod.Call(nil)
	if len(retValues) != 1 {
		return gomethods.WrongMethodSignatureError(name)
	}

	if !retValues[0].IsNil() {
		err, ok := retValues[0].Interface().(error)
		if !ok {
			return gomethods.WrongMethodSignatureError(name)
		}
		return &gomethods.MethodInvocationFailedError{MethodName: name, Err: err}
	}

	return nil
}
