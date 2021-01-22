package filter

import (
	"time"

	_ "github.com/go-sql-driver/mysql" // import driver
	"github.com/jmoiron/sqlx"
	"github.com/jpillora/backoff"
)

var MariaDB *sqlx.DB

type DBCommon interface {
	sqlx.Ext
	sqlx.Preparer
}

const jitter = 5
const minDelay = 500 * time.Millisecond
const maxDelay = 90 * time.Second
const maxAttempts = 9

func Connect(dsn string) error {
	var conn *sqlx.DB
	var err error

	bOff := backoff.Backoff{
		Min:    minDelay,
		Max:    maxDelay,
		Factor: jitter,
		Jitter: false,
	}

	for i := 0; i < maxAttempts; i++ {
		conn, err = sqlx.Connect("mysql", dsn)
		if err != nil {
			dur := bOff.Duration()
			time.Sleep(dur)

			continue
		}

		break
	}

	MariaDB = conn

	return nil
}
