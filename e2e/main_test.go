package e2e

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sharasha07/clash-bot/internal/data"
)

const apiURL = "http://localhost:8080"

func TestMain(m *testing.M) {
	defer func() {
		if err := recover(); err != nil {
			log.Fatalf("E2E test setup failed: %v", err)
		}
	}()

	dsn, ok := os.LookupEnv("TEST_DB_DSN")
	if !ok {
		panic("TEST_DB_DSN environment variable must be set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		panic(err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		panic(err)
	}

	mig, err := migrate.New("file://../migrations", dsn)
	if err != nil {
		panic(err)
	}
	defer func() {
		sourceError, dbError := mig.Close()
		if sourceError != nil || dbError != nil {
			log.Printf("migration source and database closing failed, source_error: %v, database_error: %v", sourceError, dbError)
		}
	}()
	defer func() {
		err := mig.Down()
		if err != nil {
			log.Printf("down migration failed: %v", err)
		}
	}()

	if err := mig.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		panic(err)
	}

	user := data.User{Username: "luka"}
	if err := user.Password.Set("luka123"); err != nil {
		panic(err)
	}
	if err := data.NewDBModels(pool).Users.Insert(ctx, &user); err != nil {
		panic(err)
	}

	cmd := exec.Command("../bin/api")
	cmd.Env = append(os.Environ(), "DB_DSN="+dsn, "LIMITER_ENABLED=false", "PORT=8080")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		panic(err)
	}
	defer func() {
		err := cmd.Process.Kill()
		if err != nil {
			log.Printf("killing api process failed: %v", err)
		}
	}()

	deadline := time.Now().Add(10 * time.Second)
	for {
		resp, err := http.Get(apiURL + "/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				break
			}
		}
		if time.Now().After(deadline) {
			panic("API did not become ready in time")
		}

		time.Sleep(100 * time.Millisecond)
	}

	_ = m.Run()
}
