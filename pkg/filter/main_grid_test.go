package filter

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func prepareTestData(t *testing.T) {
	setUp(t)

	_, err := MariaDB.Exec("CREATE TABLE IF NOT EXISTS file (id INTEGER PRIMARY KEY);")
	require.Nil(t, err)
	_, err = MariaDB.Exec("CREATE TABLE IF NOT EXISTS tag (id INTEGER PRIMARY KEY, file_id INTEGER, `Name` VARCHAR(255));")
	require.Nil(t, err)
	_, err = MariaDB.Exec("CREATE DATABASE IF NOT EXISTS losos;")
	require.Nil(t, err)
	_, err = MariaDB.Exec("DROP TABLE IF EXISTS losos.article;")
	require.Nil(t, err)
	_, err = MariaDB.Exec("CREATE TABLE losos.article (id INTEGER PRIMARY KEY, file_id INTEGER, `Name` VARCHAR(255));")
	require.Nil(t, err)

	_, err = MariaDB.Exec("INSERT INTO file VALUES (1);")
	require.Nil(t, err)
	_, err = MariaDB.Exec("INSERT INTO file VALUES (2);")
	require.Nil(t, err)
	_, err = MariaDB.Exec("INSERT INTO tag VALUES (1, 1, 'Losos'), (2, 1, '22');")
	require.Nil(t, err)
	_, err = MariaDB.Exec("INSERT INTO losos.article VALUES (1, 1, 'Losos'), (2, 1, '22'), (3, 1, 'Pstruh');")
	require.Nil(t, err)
}
