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
)

func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn, ok := os.LookupEnv("TEST_DB_DSN")
	if !ok {
		t.Fatal("TEST_DB_DSN environment variable must be set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}

	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}

	mig, err := migrate.New("file://../../migrations", dsn)
	if err != nil {
		t.Fatal(err)
	}

	if err := mig.Up(); err != nil {
		t.Fatal(err)
	}

	user := User{Username: "luka"}
	if err := user.SetPassword("luka123"); err != nil {
		t.Fatal(err)
	}

	if err := NewDBModels(pool).Users.Insert(ctx, &user); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if err := mig.Down(); err != nil {
			t.Errorf("migrate down: %v", err)
		}

		sourceError, dbError := mig.Close()
		if sourceError != nil || dbError != nil {
			t.Errorf("migration source and database closing failed, source_error: %v, database_error: %v", sourceError, dbError)
		}

		pool.Close()
	})

	return pool
}
