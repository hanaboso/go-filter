package filter

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func setUp(t *testing.T) func() {
	require.Nil(t, Connect("root:root@tcp(mariadb:3306)/test?parseTime=true"))

	rows, err := MariaDB.DB.Query(fmt.Sprintf("SELECT TABLE_NAME Name FROM information_schema.TABLES WHERE TABLE_SCHEMA = 'test' AND (AUTO_INCREMENT > 1 OR AUTO_INCREMENT IS NULL) AND TABLE_TYPE = 'BASE TABLE';"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = MariaDB.DB.Exec("SET FOREIGN_KEY_CHECKS=0;"); err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}

		if _, err = MariaDB.DB.Exec(fmt.Sprintf("TRUNCATE TABLE %s", name)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = MariaDB.DB.Exec("SET FOREIGN_KEY_CHECKS=1;"); err != nil {
		t.Fatal(err)
	}

	return func() {
		MariaDB.Close()
	}
}
