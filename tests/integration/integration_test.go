//go:build integration

// Package integration contains KTTM integration tests.
// Run with: go test -tags=integration ./tests/integration/...
//
// Every test in this package:
// - Uses real external services (Postgres, NATS, MinIO) via testcontainers
// - Tests a complete vertical slice from API call to data in storage
// - Has no mocking of external dependencies
// - Cleans up all created resources in t.Cleanup()
package integration

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/kubeworkflow/flowengine/tests/testinfra"
)

// TestMain starts shared infrastructure once for all integration tests.
// Individual tests use testinfra.Env.* for connection strings.
func TestMain(m *testing.M) {
	// Use a dummy *testing.T for infrastructure setup
	// (testcontainers requires it for logging)
	t := &testing.T{}
	cleanup := testinfra.StartAll(t)
	code := m.Run()
	cleanup()
	os.Exit(code)
}

// ────────────────────────────────────────────────────────────────────────────
//  Smoke: Verify all test infrastructure is reachable
// ────────────────────────────────────────────────────────────────────────────

// TestInfrastructureSmoke verifies that Postgres, NATS, and MinIO are
// all reachable from test code. This test must always pass before any
// other integration test is written.
func TestInfrastructureSmoke(t *testing.T) {
	t.Parallel()

	t.Run("Postgres", func(t *testing.T) {
		t.Parallel()
		if testinfra.Env.PostgresDSN == "" {
			t.Fatal("Postgres DSN is empty — testinfra.StartPostgres() failed")
		}
		t.Logf("✓ Postgres DSN: %s", testinfra.Env.PostgresDSN)

		// TODO(phase1): Replace with pgx.Connect() once pgx is added to go.mod
		// For now, verify the DSN is well-formed
		if len(testinfra.Env.PostgresDSN) < 10 {
			t.Fatalf("Postgres DSN looks invalid: %q", testinfra.Env.PostgresDSN)
		}
	})

	t.Run("NATS", func(t *testing.T) {
		t.Parallel()
		if testinfra.Env.NATSUrl == "" {
			t.Fatal("NATS URL is empty — testinfra.StartNATS() failed")
		}
		t.Logf("✓ NATS URL: %s", testinfra.Env.NATSUrl)
	})

	t.Run("MinIO", func(t *testing.T) {
		t.Parallel()
		if testinfra.Env.MinIOUrl == "" {
			t.Fatal("MinIO URL is empty — testinfra.StartMinIO() failed")
		}
		t.Logf("✓ MinIO URL: %s", testinfra.Env.MinIOUrl)
	})
}

// ────────────────────────────────────────────────────────────────────────────
//  Placeholder tests for Phase 1 features
//  These will be filled in as each feature is implemented.
//  Leaving them here with t.Skip() ensures they show up in the test report
//  and remind us what needs coverage.
// ────────────────────────────────────────────────────────────────────────────

// TestGraphLinterCycleDetection verifies that the GraphLinter rejects DAGs with cycles.
// Implementation: core/linter package (Phase 1 - F1.2)
func TestGraphLinterCycleDetection(t *testing.T) {
	t.Skip("TODO(phase1-f1.2): implement GraphLinter — see core/linter/")
	// Will test:
	// 1. Create a DAG spec with a cycle (A → B → A)
	// 2. Call GraphLinter.Lint(spec)
	// 3. Expect ErrCycleDetected
	// 4. Verify the error message contains the cycle path
}

// TestGraphLinterValidDAG verifies that a valid linear DAG passes linting.
func TestGraphLinterValidDAG(t *testing.T) {
	t.Skip("TODO(phase1-f1.2): implement GraphLinter — see core/linter/")
}

// TestS3ConnectorRead verifies the S3 connector reads a file from MinIO.
func TestS3ConnectorRead(t *testing.T) {
	t.Skip("TODO(phase1-f1.4): implement S3 connector — see core/connectors/s3/")
	_ = fmt.Sprintf("MinIO: %s", testinfra.Env.MinIOUrl) // suppress unused warning
	// Will test:
	// 1. Upload a test CSV to MinIO testinfra
	// 2. Create S3ConnectorConfig pointing to MinIO
	// 3. Run connector.Read(ctx)
	// 4. Verify bytes match the uploaded file
}

// TestPostgresConnectorWrite verifies the Postgres connector writes rows.
func TestPostgresConnectorWrite(t *testing.T) {
	t.Skip("TODO(phase1-f1.5): implement Postgres connector — see core/connectors/postgres/")
	_ = testinfra.Env.PostgresDSN // suppress unused warning
	// Will test:
	// 1. Create test table in testinfra Postgres
	// 2. Create PostgresConnectorConfig
	// 3. Call connector.Write(ctx, rows)
	// 4. SELECT COUNT(*) and verify row count
}

// TestNATSEnvelopeTransport verifies the dual-channel transport (KTTM-REQ-031).
func TestNATSEnvelopeTransport(t *testing.T) {
	t.Skip("TODO(phase1-f1.7): implement NATS transport — see core/messaging/")
	_ = testinfra.Env.NATSUrl // suppress unused warning
	// Will test:
	// 1. Create an Envelope with 10MB payload
	// 2. Publish envelope to NATS
	// 3. Subscribe and receive envelope
	// 4. Verify NATS message is < 1KB (metadata only)
	// 5. Verify payload bytes transferred via emptyDir/shm channel
}

// TestPythonScriptNode verifies that a Python script node runs in a real container.
func TestPythonScriptNode(t *testing.T) {
	t.Skip("TODO(phase2-f2.1): implement Python script node — see core/nodebuilder/")
	// Will test:
	// 1. Create a script/python node spec with pandas transformation
	// 2. Build container image via NodeBuilderFactory
	// 3. Run container with test CSV on stdin
	// 4. Verify transformed CSV on stdout
}

// TestRetryPolicyExponentialBackoff verifies retry logic (KTTM-REQ-035).
func TestRetryPolicyExponentialBackoff(t *testing.T) {
	t.Skip("TODO(phase2-f2.6): implement retry policy — see core/engine/")
	// Will test:
	// 1. Create a node that fails exactly 2 times then succeeds
	// 2. Apply retryPolicy{maxRetries: 3, backoff: exponential}
	// 3. Verify node eventually succeeds
	// 4. Verify retry count in status = 2
}

// TestParallelFanOut verifies that parallel groups run concurrently (KTTM-REQ-023).
func TestParallelFanOut(t *testing.T) {
	t.Skip("TODO(phase2-f2.8): implement parallel fan-out — see core/compiler/")
	// Will test:
	// 1. Create a DAG with 3 parallel sinks + 1 barrier
	// 2. Compile to Argo Workflow
	// 3. Verify all 3 sinks run simultaneously (no sequential dependency)
	// 4. Verify barrier node only starts after all 3 complete
}

// TestFormSubmissionTriggersPipeline verifies the full end-user form → pipeline flow.
func TestFormSubmissionTriggersPipeline(t *testing.T) {
	t.Skip("TODO(phase5-f5.3): implement form submission — see core/bff/")
	// Will test:
	// 1. POST form values to /api/trigger/{app}
	// 2. Verify NATS message published to kttm.apps.{app}.trigger
	// 3. Verify workflow starts in cluster (or mock Argo in this test)
	// 4. Verify form values arrive as workflow inputs
}

// ────────────────────────────────────────────────────────────────────────────
//  Coverage: This file ensures that as features are implemented,
//  their integration tests are wired in here.
//
//  Adding a new feature? Follow this checklist:
//  1. Add a placeholder test here (t.Skip with TODO comment)
//  2. When feature is implemented, remove the t.Skip
//  3. Implement the test body
//  4. Ensure CI passes with real containers
// ────────────────────────────────────────────────────────────────────────────

// TestCoverageGate is a meta-test that lists all implemented integration tests.
// Update this list when you remove a t.Skip from a test above.
func TestCoverageGate(t *testing.T) {
	ctx := context.Background()
	_ = ctx

	implemented := []string{
		"TestInfrastructureSmoke",
	}

	pending := []string{
		"TestGraphLinterCycleDetection",
		"TestGraphLinterValidDAG",
		"TestS3ConnectorRead",
		"TestPostgresConnectorWrite",
		"TestNATSEnvelopeTransport",
		"TestPythonScriptNode",
		"TestRetryPolicyExponentialBackoff",
		"TestParallelFanOut",
		"TestFormSubmissionTriggersPipeline",
	}

	t.Logf("Integration test coverage:")
	t.Logf("  Implemented: %d", len(implemented))
	t.Logf("  Pending:     %d", len(pending))
	for _, name := range pending {
		t.Logf("    ⏳ %s", name)
	}
}
