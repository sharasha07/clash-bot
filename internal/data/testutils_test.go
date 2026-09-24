package data

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn, ok := os.LookupEnv("TEST_DB_DSN")
	if !ok {
		t.Fatal("TEST_DB_DSN environment variable must be set")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	err = pool.Ping(ctx)
	require.NoError(t, err)

	mig, err := migrate.New("file://../../migrations", dsn)
	require.NoError(t, err)
	t.Cleanup(func() {
		if err := mig.Down(); err != nil {
			t.Errorf("migrate down: %v", err)
		}

		sourceError, dbError := mig.Close()
		if sourceError != nil || dbError != nil {
			t.Errorf("migration source and database closing failed, source_error: %v, database_error: %v", sourceError, dbError)
		}
	})

	err = mig.Up()
	require.NoError(t, err)

	user := User{Username: "luka"}
	err = user.SetPassword("luka123")
	require.NoError(t, err)

	err = NewModels(pool).Users.Insert(ctx, &user)
	require.NoError(t, err)

	return pool
}
