package e2e

import (
	"context"
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

const (
	port   = "8080"
	apiURL = "http://localhost:" + port
)

func TestMain(m *testing.M) {
	var exitCode int

	func() {
		// for simpler error message
		defer func() {
			if err := recover(); err != nil {
				log.Fatalf("E2E test setup failed: %v", err)
			}
		}()

		// database setup for tests
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

		if err := mig.Up(); err != nil {
			panic(err)
		}

		user := data.User{Username: "nika"}
		if err := user.SetPassword("nika123"); err != nil {
			panic(err)
		}

		if err := data.NewModels(pool).Users.Insert(ctx, &user); err != nil {
			panic(err)
		}

		// starting API process for tests
		cmd := exec.Command("../bin/api")
		cmd.Env = append(os.Environ(), "DB_DSN="+dsn, "LIMITER_ENABLED=false", "PORT="+port)
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

			err = cmd.Wait()
			if err != nil {
				log.Println(err)
			}
		}()

		deadline := time.Now().Add(10 * time.Second)
		for {
			client := http.Client{Timeout: 3 * time.Second}
			resp, err := client.Get(apiURL + "/health")
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

		// running tests
		exitCode = m.Run()
	}()

	os.Exit(exitCode)
}
