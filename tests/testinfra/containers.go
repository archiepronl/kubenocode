// Package testinfra provides shared test infrastructure for all KTTM test suites.
//
// This package spins up real backing services using testcontainers-go.
// No mocking for external services — we test against real Postgres, NATS, and MinIO.
//
// Usage in integration tests:
//
//	func TestMain(m *testing.M) {
//	    testinfra.StartAll()
//	    code := m.Run()
//	    testinfra.StopAll()
//	    os.Exit(code)
//	}
package testinfra

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Env holds all connection strings and addresses for the test infrastructure.
// Populated by Start* functions. Safe to read after StartAll().
var Env = struct {
	PostgresDSN string
	NATSUrl     string
	MinIOUrl    string
	MinIOUser   string
	MinIOPass   string
}{}

// StartAll starts Postgres, NATS JetStream, and MinIO.
// Call in TestMain for integration test suites.
func StartAll(ctx context.Context) (func(), error) {
	cleanups := make([]func(), 0, 3)
	cleanup := func() {
		for i := len(cleanups) - 1; i >= 0; i-- {
			cleanups[i]()
		}
	}
	start := []func(context.Context) (func(), error){StartPostgres, StartNATS, StartMinIO}
	for _, startService := range start {
		stop, err := startService(ctx)
		if err != nil {
			cleanup()
			return nil, err
		}
		cleanups = append(cleanups, stop)
	}
	return cleanup, nil
}

// StartPostgres starts a real PostgreSQL 16 container.
// Returns a cleanup function — call it when the test suite finishes.
func StartPostgres(ctx context.Context) (func(), error) {
	if dsn := os.Getenv("POSTGRES_DSN"); dsn != "" {
		Env.PostgresDSN = dsn
		log.Printf("testinfra: using external Postgres")
		return func() {}, nil
	}

	log.Print("testinfra: starting PostgreSQL 16 container...")
	c, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:16-alpine"),
		postgres.WithDatabase("kttm_test"),
		postgres.WithUsername("kttm"),
		postgres.WithPassword("kttm_test_secret"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("testinfra: failed to start Postgres: %w", err)
	}

	dsn, err := c.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = c.Terminate(context.Background())
		return nil, fmt.Errorf("testinfra: failed to get Postgres DSN: %w", err)
	}

	Env.PostgresDSN = dsn
	log.Print("testinfra: Postgres ready")

	return func() {
		if err := c.Terminate(ctx); err != nil {
			log.Printf("testinfra: Postgres cleanup error: %v", err)
		}
	}, nil
}

// StartNATS starts a real NATS JetStream container.
func StartNATS(ctx context.Context) (func(), error) {
	if url := os.Getenv("NATS_URL"); url != "" {
		Env.NATSUrl = url
		log.Printf("testinfra: using external NATS")
		return func() {}, nil
	}

	log.Print("testinfra: starting NATS JetStream container...")
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "nats:2.10-alpine",
			ExposedPorts: []string{"4222/tcp"},
			Cmd:          []string{"-js", "-sd", "/tmp/jetstream"},
			WaitingFor:   wait.ForLog("Server is ready").WithStartupTimeout(30 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		return nil, fmt.Errorf("testinfra: failed to start NATS: %w", err)
	}
	host, err := c.Host(ctx)
	if err != nil {
		_ = c.Terminate(context.Background())
		return nil, fmt.Errorf("testinfra: failed to get NATS host: %w", err)
	}
	port, err := c.MappedPort(ctx, "4222/tcp")
	if err != nil {
		_ = c.Terminate(context.Background())
		return nil, fmt.Errorf("testinfra: failed to get NATS port: %w", err)
	}
	url := fmt.Sprintf("nats://%s:%s", host, port.Port())
	Env.NATSUrl = url
	log.Printf("testinfra: NATS ready at %s", url)

	return func() {
		if err := c.Terminate(ctx); err != nil {
			log.Printf("testinfra: NATS cleanup error: %v", err)
		}
	}, nil
}

// StartMinIO starts a real S3-compatible object store container.
func StartMinIO(ctx context.Context) (func(), error) {
	if url := os.Getenv("MINIO_URL"); url != "" {
		Env.MinIOUrl = url
		Env.MinIOUser = os.Getenv("MINIO_USER")
		Env.MinIOPass = os.Getenv("MINIO_PASS")
		log.Printf("testinfra: using external MinIO")
		return func() {}, nil
	}

	log.Print("testinfra: starting S3-compatible object store container...")
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "motoserver/moto:5.0.28",
			ExposedPorts: []string{"5000/tcp"},
			WaitingFor:   wait.ForListeningPort("5000/tcp").WithStartupTimeout(60 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		return nil, fmt.Errorf("testinfra: failed to start S3-compatible object store: %w", err)
	}

	host, err := c.Host(ctx)
	if err != nil {
		_ = c.Terminate(context.Background())
		return nil, fmt.Errorf("testinfra: failed to get object store host: %w", err)
	}
	port, err := c.MappedPort(ctx, "5000/tcp")
	if err != nil {
		_ = c.Terminate(context.Background())
		return nil, fmt.Errorf("testinfra: failed to get object store port: %w", err)
	}

	Env.MinIOUrl = fmt.Sprintf("http://%s:%s", host, port.Port())
	Env.MinIOUser = "kttmtest"
	Env.MinIOPass = "kttmtest123"
	log.Printf("testinfra: S3-compatible object store ready at %s", Env.MinIOUrl)

	return func() {
		if err := c.Terminate(ctx); err != nil {
			log.Printf("testinfra: S3-compatible object store cleanup error: %v", err)
		}
	}, nil
}

// SkipIfNoDocker skips the test if Docker is not available (useful in unit test suites).
func SkipIfNoDocker(t *testing.T) {
	t.Helper()
	if os.Getenv("DOCKER_HOST") == "" && !dockerAvailable() {
		t.Skip("skipping: Docker not available (set DOCKER_HOST or start Docker Desktop)")
	}
}

func dockerAvailable() bool {
	// Try to connect to Docker daemon
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req := testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image: "alpine:latest",
			Cmd:   []string{"echo", "ok"},
		},
		Started: false,
	}
	_, err := testcontainers.GenericContainer(ctx, req)
	return err == nil
}
