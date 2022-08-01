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

func WrapErrFailedToSendCloseNotify(err error) error {
	//https://github.com/jackc/pgx/issues/984
	//In Azure Pgx is throwing an error while the close is called on a connection.
	//The code below will wrap a custom error type, which will allow the consumer to ignore it.
	if strings.Contains(err.Error(), "failed to send closeNotify alert (but connection was closed anyway)") {
		return fmt.Errorf(err.Error()+": %w", ErrFailedToSendCloseNotify)
	}
	return err
}
