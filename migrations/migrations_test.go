package migrations_test

import (
	"io/fs"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MaxCaribe/library-go/migrations"
)

// These run without a database. They catch the mistakes that otherwise only
// surface when a migration is applied: a duplicate version, a name goose
// cannot parse, or a missing Down section.
var migrationName = regexp.MustCompile(`^\d{4,}_[a-z0-9_]+\.sql$`)

func TestMigrationsAreWellFormed(t *testing.T) {
	files := migrationFiles(t)
	require.NotEmpty(t, files, "no migrations are embedded")

	versions := map[int]string{}
	for _, name := range files {
		t.Run(name, func(t *testing.T) {
			require.Regexp(t, migrationName, name, "must be named <version>_<snake_case>.sql")

			version, err := strconv.Atoi(strings.SplitN(name, "_", 2)[0])
			require.NoError(t, err)

			previous, duplicate := versions[version]
			require.False(t, duplicate, "version %d is already used by %s", version, previous)
			versions[version] = name

			body := readMigration(t, name)
			assert.Contains(t, body, "-- +goose Up", "missing goose Up section")
			assert.Contains(t, body, "-- +goose Down", "missing goose Down section, so it cannot be rolled back")
		})
	}
}

func TestFirstMigrationCreatesBooks(t *testing.T) {
	body := readMigration(t, "0001_create_books.sql")

	assert.Contains(t, body, "CREATE TABLE books")
	assert.Contains(t, body, "authors      TEXT[] NOT NULL", "authors are stored as an ordered array, not a child table")
}

func migrationFiles(t *testing.T) []string {
	t.Helper()

	files, err := fs.Glob(migrations.FS, "*.sql")
	require.NoError(t, err)
	return files
}

func readMigration(t *testing.T, name string) string {
	t.Helper()

	body, err := migrations.FS.ReadFile(name)
	require.NoError(t, err)
	return string(body)
}
