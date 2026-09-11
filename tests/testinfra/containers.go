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
	"os"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/minio"
	"github.com/testcontainers/testcontainers-go/modules/nats"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Env holds all connection strings and addresses for the test infrastructure.
// Populated by Start* functions. Safe to read after StartAll().
var Env = struct {
	PostgresDSN  string
	NATSUrl      string
	MinIOUrl     string
	MinIOUser    string
	MinIOPass    string
}{}

var (
	pgContainer    *postgres.PostgresContainer
	natsContainer  *nats.NATSContainer
	minioContainer *minio.MinioContainer
	cancelFuncs    []context.CancelFunc
)

// StartAll starts Postgres, NATS JetStream, and MinIO.
// Call in TestMain for integration test suites.
func StartAll(t *testing.T) func() {
	t.Helper()
	ctx := context.Background()

	cleanup1 := StartPostgres(t, ctx)
	cleanup2 := StartNATS(t, ctx)
	cleanup3 := StartMinIO(t, ctx)

	return func() {
		cleanup3()
		cleanup2()
		cleanup1()
	}
}

// StartPostgres starts a real PostgreSQL 16 container.
// Returns a cleanup function — call it when the test suite finishes.
func StartPostgres(t *testing.T, ctx context.Context) func() {
	t.Helper()

	if dsn := os.Getenv("POSTGRES_DSN"); dsn != "" {
		Env.PostgresDSN = dsn
		t.Logf("testinfra: using external Postgres: %s", dsn)
		return func() {}
	}

	t.Log("testinfra: starting PostgreSQL 16 container...")
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
		t.Fatalf("testinfra: failed to start Postgres: %v", err)
	}

	dsn, err := c.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("testinfra: failed to get Postgres DSN: %v", err)
	}

	pgContainer = c
	Env.PostgresDSN = dsn
	t.Logf("testinfra: Postgres ready: %s", dsn)

	return func() {
		if err := c.Terminate(ctx); err != nil {
			t.Logf("testinfra: Postgres cleanup error: %v", err)
		}
	}
}

// StartNATS starts a real NATS JetStream container.
func StartNATS(t *testing.T, ctx context.Context) func() {
	t.Helper()

	if url := os.Getenv("NATS_URL"); url != "" {
		Env.NATSUrl = url
		t.Logf("testinfra: using external NATS: %s", url)
		return func() {}
	}

	t.Log("testinfra: starting NATS JetStream container...")
	c, err := nats.RunContainer(ctx,
		testcontainers.WithImage("nats:2.10-alpine"),
		nats.WithArgument("jetstream"),
		nats.WithArgument("store_dir", "/tmp/jetstream"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("Server is ready").WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("testinfra: failed to start NATS: %v", err)
	}

	url, err := c.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("testinfra: failed to get NATS URL: %v", err)
	}

	natsContainer = c
	Env.NATSUrl = url
	t.Logf("testinfra: NATS ready: %s", url)

	return func() {
		if err := c.Terminate(ctx); err != nil {
			t.Logf("testinfra: NATS cleanup error: %v", err)
		}
	}
}

// StartMinIO starts a real MinIO (S3-compatible) container.
func StartMinIO(t *testing.T, ctx context.Context) func() {
	t.Helper()

	if url := os.Getenv("MINIO_URL"); url != "" {
		Env.MinIOUrl = url
		Env.MinIOUser = os.Getenv("MINIO_USER")
		Env.MinIOPass = os.Getenv("MINIO_PASS")
		t.Logf("testinfra: using external MinIO: %s", url)
		return func() {}
	}

	t.Log("testinfra: starting MinIO container...")
	c, err := minio.RunContainer(ctx,
		testcontainers.WithImage("minio/minio:latest"),
		minio.WithUsername("kttmtest"),
		minio.WithPassword("kttmtest123"),
	)
	if err != nil {
		t.Fatalf("testinfra: failed to start MinIO: %v", err)
	}

	url, err := c.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("testinfra: failed to get MinIO URL: %v", err)
	}

	minioContainer = c
	Env.MinIOUrl = fmt.Sprintf("http://%s", url)
	Env.MinIOUser = "kttmtest"
	Env.MinIOPass = "kttmtest123"
	t.Logf("testinfra: MinIO ready: %s", Env.MinIOUrl)

	return func() {
		if err := c.Terminate(ctx); err != nil {
			t.Logf("testinfra: MinIO cleanup error: %v", err)
		}
	}
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
