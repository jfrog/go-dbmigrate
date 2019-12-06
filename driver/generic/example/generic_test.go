package example

import (
	"database/sql"
	"github.com/jfrog/go-dbmigrate/driver"
	"testing"

	"github.com/jfrog/go-dbmigrate/file"
	"github.com/jfrog/go-dbmigrate/migrate/direction"

	"github.com/jfrog/go-dbmigrate/driver/generic"
	"github.com/jfrog/go-dbmigrate/driver/mongodb/gomethods"
	pipep "github.com/jfrog/go-dbmigrate/pipe"
	_ "github.com/lib/pq"
	"os"
	"reflect"
	"time"
)

type ExpectedMigrationResult struct {
	Organizations    []Organization
	Organizations_v2 []Organization_v2
	Users            []User
	Errors           []error
}

func RunMigrationAndAssertResult(
	t *testing.T,
	title string,
	d *generic.Driver,
	file file.File,
	expected *ExpectedMigrationResult) {

	actualOrganizations := []Organization{}
	actualOrganizations_v2 := []Organization_v2{}
	actualUsers := []User{}
	var err error
	var pipe chan interface{}
	var errs []error

	pipe = pipep.New()
	go d.Migrate(file, pipe)
	errs = pipep.ReadErrors(pipe)

	session, err := getMongoSession()
	if err != nil {
		t.Fatal("Failed to open mongo session: $v", err)
	}
	defer session.Close()
	if len(expected.Organizations) > 0 {
		err = session.DB(DB_NAME).C(ORGANIZATIONS_C).Find(nil).All(&actualOrganizations)
	} else {
		err = session.DB(DB_NAME).C(ORGANIZATIONS_C).Find(nil).All(&actualOrganizations_v2)
	}
	if err != nil {
		t.Fatal("Failed to query Organizations collection")
	}

	err = session.DB(DB_NAME).C(USERS_C).Find(nil).All(&actualUsers)
	if err != nil {
		t.Fatal("Failed to query Users collection")
	}

	if !reflect.DeepEqual(expected.Errors, errs) {
		t.Errorf("Migration '%s': FAILED\nexpected errors %v\nbut got %v", title, expected.Errors, errs)
	}

	if !reflect.DeepEqual(expected.Organizations, actualOrganizations) {
		t.Errorf("Migration '%s': FAILED\nexpected organizations %v\nbut got %v", title, expected.Organizations, actualOrganizations)
	}

	if !reflect.DeepEqual(expected.Organizations_v2, actualOrganizations_v2) {
		t.Errorf("Migration '%s': FAILED\nexpected organizations v2 %v\nbut got %v", title, expected.Organizations_v2, actualOrganizations_v2)
	}

	if !reflect.DeepEqual(expected.Users, actualUsers) {
		t.Errorf("Migration '%s': FAILED\nexpected users %v\nbut got %v", title, expected.Users, actualUsers)

	}
	// t.Logf("Migration '%s': PASSED", title)
}

func TestMigrate(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Test failed on panic: %v", r)
		}
	}()

	host := os.Getenv("POSTGRES_PORT_5432_TCP_ADDR")
	port := os.Getenv("POSTGRES_PORT_5432_TCP_PORT")
	migrationsDbUrl := "postgres://postgres@" + host + ":" + port + "/template1?sslmode=disable"
	driverUrl := "generic://postgres@" + host + ":" + port + "/template1?sslmode=disable&migrations_db_type=postgres"

	// prepare clean database
	connection, err := sql.Open("postgres", migrationsDbUrl)
	if err != nil {
		t.Fatal(err)
	}

	// Reset DB
	if _, err := connection.Exec(`
				DROP TABLE IF EXISTS db_migrations;`); err != nil {
		t.Fatal(err)
	}
	session, err := getMongoSession()
	if err != nil {
		t.Fatalf("could not get mongo session: %v", err)
	}
	defer session.Close()
	session.DB(DB_NAME).C(ORGANIZATIONS_C).DropCollection()
	session.DB(DB_NAME).C(USERS_C).DropCollection()

	gen, ok := driver.GetDriverGenerator("generic")
	if !ok {
		t.Fatal("Generic driver has not been registered")
	}
	d0 := gen.Generate()
	d, ok := d0.(*generic.Driver)
	if !ok {
		t.Fatal("MongoDbGoMethodsDriver has not been registered")
	}
	if err := d.Initialize(driverUrl); err != nil {
		t.Fatal(err)
	}

	date1, _ := time.Parse(SHORT_DATE_LAYOUT, "1994-Jul-05")
	date2, _ := time.Parse(SHORT_DATE_LAYOUT, "1998-Sep-04")
	date3, _ := time.Parse(SHORT_DATE_LAYOUT, "2008-Apr-28")

	migrations := []struct {
		name           string
		file           file.File
		expectedResult ExpectedMigrationResult
	}{
		{
			name: "v0 -> v1",
			file: file.File{
				Path:      "/foobar",
				FileName:  "001_foobar.up.gm",
				Version:   1,
				Name:      "foobar",
				Direction: direction.Up,
				Content: []byte(`
						V001_init_organizations_up
						V001_init_users_up
					`),
			},
			expectedResult: ExpectedMigrationResult{
				Organizations: []Organization{
					{Id: OrganizationIds[0], Name: "Amazon", Location: "Seattle", DateFounded: date1},
					{Id: OrganizationIds[1], Name: "Google", Location: "Mountain View", DateFounded: date2},
					{Id: OrganizationIds[2], Name: "JFrog", Location: "Santa Clara", DateFounded: date3},
				},
				Organizations_v2: []Organization_v2{},
				Users: []User{
					{Id: UserIds[0], Name: "Alex"},
					{Id: UserIds[1], Name: "Beatrice"},
					{Id: UserIds[2], Name: "Cleo"},
				},
				Errors: []error{},
			},
		},
		{
			name: "v1 -> v2",
			file: file.File{
				Path:      "/foobar",
				FileName:  "002_foobar.up.gm",
				Version:   2,
				Name:      "foobar",
				Direction: direction.Up,
				Content: []byte(`
						V002_organizations_rename_location_field_to_headquarters_up
						V002_change_user_cleo_to_cleopatra_up
					`),
			},
			expectedResult: ExpectedMigrationResult{
				Organizations: []Organization{},
				Organizations_v2: []Organization_v2{
					{Id: OrganizationIds[0], Name: "Amazon", Headquarters: "Seattle", DateFounded: date1},
					{Id: OrganizationIds[1], Name: "Google", Headquarters: "Mountain View", DateFounded: date2},
					{Id: OrganizationIds[2], Name: "JFrog", Headquarters: "Santa Clara", DateFounded: date3},
				},
				Users: []User{
					{Id: UserIds[0], Name: "Alex"},
					{Id: UserIds[1], Name: "Beatrice"},
					{Id: UserIds[2], Name: "Cleopatra"},
				},
				Errors: []error{},
			},
		},
		{
			name: "v2 -> v1",
			file: file.File{
				Path:      "/foobar",
				FileName:  "002_foobar.down.gm",
				Version:   2,
				Name:      "foobar",
				Direction: direction.Down,
				Content: []byte(`
						V002_change_user_cleo_to_cleopatra_down
						V002_organizations_rename_location_field_to_headquarters_down
					`),
			},
			expectedResult: ExpectedMigrationResult{
				Organizations: []Organization{
					{Id: OrganizationIds[0], Name: "Amazon", Location: "Seattle", DateFounded: date1},
					{Id: OrganizationIds[1], Name: "Google", Location: "Mountain View", DateFounded: date2},
					{Id: OrganizationIds[2], Name: "JFrog", Location: "Santa Clara", DateFounded: date3},
				},
				Organizations_v2: []Organization_v2{},
				Users: []User{
					{Id: UserIds[0], Name: "Alex"},
					{Id: UserIds[1], Name: "Beatrice"},
					{Id: UserIds[2], Name: "Cleo"},
				},
				Errors: []error{},
			},
		},
		{
			name: "v1 -> v0",
			file: file.File{
				Path:      "/foobar",
				FileName:  "001_foobar.down.gm",
				Version:   1,
				Name:      "foobar",
				Direction: direction.Down,
				Content: []byte(`
						V001_init_users_down
						V001_init_organizations_down
					`),
			},
			expectedResult: ExpectedMigrationResult{
				Organizations:    []Organization{},
				Organizations_v2: []Organization_v2{},
				Users:            []User{},
				Errors:           []error{},
			},
		},
		{
			name: "v0 -> v1: missing method aborts migration",
			file: file.File{
				Path:      "/foobar",
				FileName:  "001_foobar.up.gm",
				Version:   1,
				Name:      "foobar",
				Direction: direction.Up,
				Content: []byte(`
						V001_init_organizations_up
						V001_init_users_up
						v001_non_existing_method_up
					`),
			},
			expectedResult: ExpectedMigrationResult{
				Organizations:    []Organization{},
				Organizations_v2: []Organization_v2{},
				Users:            []User{},
				Errors:           []error{gomethods.MissingMethodError("v001_non_existing_method_up")},
			},
		},
		{
			name: "v0 -> v1: not exported method aborts migration",
			file: file.File{
				Path:      "/foobar",
				FileName:  "001_foobar.up.gm",
				Version:   1,
				Name:      "foobar",
				Direction: direction.Up,
				Content: []byte(`
						V001_init_organizations_up
						v001_not_exported_method_up
						V001_init_users_up
					`),
			},
			expectedResult: ExpectedMigrationResult{
				Organizations:    []Organization{},
				Organizations_v2: []Organization_v2{},
				Users:            []User{},
				//Errors:           []error{m.MethodNotExportedError("v001_not_exported_method_up")},
				Errors: []error{gomethods.MissingMethodError("v001_not_exported_method_up")},
			},
		},
		{
			name: "v0 -> v1: wrong signature method aborts migration",
			file: file.File{
				Path:      "/foobar",
				FileName:  "001_foobar.up.gm",
				Version:   1,
				Name:      "foobar",
				Direction: direction.Up,
				Content: []byte(`
						V001_init_organizations_up
						V001_method_with_wrong_signature_up
						V001_init_users_up
					`),
			},
			expectedResult: ExpectedMigrationResult{
				Organizations:    []Organization{},
				Organizations_v2: []Organization_v2{},
				Users:            []User{},
				Errors:           []error{gomethods.WrongMethodSignatureError("V001_method_with_wrong_signature_up")},
			},
		},
		{
			name: "v1 -> v0: wrong signature method aborts migration",
			file: file.File{
				Path:      "/foobar",
				FileName:  "001_foobar.down.gm",
				Version:   1,
				Name:      "foobar",
				Direction: direction.Down,
				Content: []byte(`
						V001_init_users_down
						V001_method_with_wrong_signature_down
						V001_init_organizations_down
					`),
			},
			expectedResult: ExpectedMigrationResult{
				Organizations:    []Organization{},
				Organizations_v2: []Organization_v2{},
				Users:            []User{},
				Errors:           []error{gomethods.WrongMethodSignatureError("V001_method_with_wrong_signature_down")},
			},
		},
	}

	for _, m := range migrations {
		RunMigrationAndAssertResult(t, m.name, d, m.file, &m.expectedResult)
	}

	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
}
