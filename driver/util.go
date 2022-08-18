package driver

import (
	"fmt"
	"hash/crc32"
	"strings"
)

const advisoryLockIDSalt uint = 1486364155

// GenerateAdvisoryLockId inspired by rails migrations, see https://goo.gl/8o9bCT
func GenerateAdvisoryLockId(databaseName string, additionalNames ...string) (string, error) { // nolint: golint
	if len(additionalNames) > 0 {
		databaseName = strings.Join(append(additionalNames, databaseName), "\x00")
	}
	sum := crc32.ChecksumIEEE([]byte(databaseName))
	sum = sum * uint32(advisoryLockIDSalt)
	return fmt.Sprint(sum), nil
}

func CanIgnoreError(err error) bool {
	if err == nil {
		return true
	}
	//https://github.com/jackc/pgx/issues/984
	//In Azure, Pgx is throwing an error while closing the connection.
	//We are ignoring this error, as the connection was closed anyway.
	if strings.Contains(err.Error(), "failed to send closeNotify alert (but connection was closed anyway)") {
		return true
	}

	return false
}
