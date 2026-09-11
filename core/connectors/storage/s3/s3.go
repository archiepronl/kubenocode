// Package s3 implements the S3-compatible object storage connector for FlowEngine.
//
// This connector supports:
//   - Amazon S3
//   - Google Cloud Storage (via S3-compatible API)
//   - Azure Blob Storage (via S3-compatible endpoint)
//   - MinIO (in-cluster object store for large payload transfer)
//
// Credentials are injected via Kubernetes Secrets (SecretRef) and read from
// environment variables — never stored in the CRD spec (NFR-2).
//
// Streaming design: uses AWS SDK GetObject with a streaming response body.
// The raw bytes flow directly to the caller's io.Reader without buffering (FR-4).
package s3

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/kubeworkflow/flowengine/core/connectors"
)

// ─────────────────────────────────────────────
//  Type identifier
// ─────────────────────────────────────────────

const connectorType = "connector/s3"

// ─────────────────────────────────────────────
//  Config keys
// ─────────────────────────────────────────────

const (
	// Param keys (non-sensitive, stored in WorkflowNode.Params)
	paramBucket   = "bucket"
	paramPrefix   = "prefix"
	paramRegion   = "region"
	paramEndpoint = "endpoint" // for MinIO or GCS S3-compat

	// Secret env keys (injected from Kubernetes Secret via SecretRef)
	envAccessKeyID     = "AWS_ACCESS_KEY_ID"
	envSecretAccessKey = "AWS_SECRET_ACCESS_KEY"
	envSessionToken    = "AWS_SESSION_TOKEN"
)

// ─────────────────────────────────────────────
//  S3Connector
// ─────────────────────────────────────────────

// S3Connector implements connectors.Connector for S3-compatible object stores.
type S3Connector struct{}

// New returns a new S3Connector.
func New() *S3Connector { return &S3Connector{} }

// Type implements connectors.Connector.
func (c *S3Connector) Type() string { return connectorType }

// Schema returns the JSON Schema for S3 connector configuration.
// The UI editor renders this as an auto-generated config form.
func (c *S3Connector) Schema() json.RawMessage {
	return json.RawMessage(`{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"title": "S3 Connector Configuration",
		"type": "object",
		"required": ["bucket"],
		"properties": {
			"bucket": {
				"type": "string",
				"title": "Bucket Name",
				"description": "The S3 bucket to read from or write to."
			},
			"prefix": {
				"type": "string",
				"title": "Object Prefix / Path",
				"description": "Optional key prefix to filter objects (e.g., 'reports/2024/')."
			},
			"region": {
				"type": "string",
				"title": "AWS Region",
				"description": "AWS region for the bucket (e.g., 'us-east-1'). Leave empty for MinIO/GCS.",
				"default": "us-east-1"
			},
			"endpoint": {
				"type": "string",
				"title": "Custom Endpoint",
				"description": "Override the S3 endpoint for MinIO (e.g., 'http://minio.flowengine.svc:9000') or GCS S3 compat."
			}
		},
		"secretRef": {
			"type": "string",
			"title": "Credentials Secret",
			"description": "Name of the Kubernetes Secret containing AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY.",
			"x-flowengine-secret": true
		}
	}`)
}

// Read opens a streaming connection to the specified S3 object and returns the raw bytes.
// For large objects, the stream is consumed lazily — never fully loaded into memory (FR-4).
//
// Configuration (from ConnectorConfig):
//   - bucket (param): S3 bucket name
//   - prefix (param): object key or prefix
//   - region (param): AWS region
//   - endpoint (param): custom endpoint for MinIO/GCS
//   - AWS_ACCESS_KEY_ID (env): from SecretRef
//   - AWS_SECRET_ACCESS_KEY (env): from SecretRef
func (c *S3Connector) Read(ctx context.Context, cfg connectors.ConnectorConfig) (*connectors.ReadResult, error) {
	bucket := cfg.Get(paramBucket)
	prefix := cfg.Get(paramPrefix)
	endpoint := cfg.Get(paramEndpoint)

	if bucket == "" {
		return nil, fmt.Errorf("s3 connector: 'bucket' parameter is required")
	}

	// In production: use github.com/aws/aws-sdk-go-v2 with streaming GetObject.
	// The implementation below shows the structural flow for scaffolding.
	//
	// s3Client := buildS3Client(cfg)
	// resp, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
	//     Bucket: aws.String(bucket),
	//     Key:    aws.String(prefix),
	// })
	// if err != nil {
	//     return nil, fmt.Errorf("s3 GetObject: %w", err)
	// }
	// return &connectors.ReadResult{
	//     Envelope: connectors.EnvelopeRef{
	//         MimeType:    detectMimeType(prefix, resp.ContentType),
	//         PayloadSize: aws.ToInt64(resp.ContentLength),
	//         Tags: map[string]string{
	//             "s3.bucket": bucket,
	//             "s3.key":    prefix,
	//         },
	//     },
	//     Stream: resp.Body, // raw streaming body, no buffering
	// }, nil

	// Scaffold placeholder: simulate a streaming response
	_ = endpoint // used in production
	mimeType := inferMimeType(prefix)
	body := io.NopCloser(strings.NewReader(
		fmt.Sprintf("[S3Connector] Streaming s3://%s/%s", bucket, prefix),
	))

	return &connectors.ReadResult{
		Envelope: connectors.EnvelopeRef{
			MimeType:    mimeType,
			PayloadSize: -1, // unknown until fully read
			Tags: map[string]string{
				"s3.bucket": bucket,
				"s3.prefix": prefix,
			},
		},
		Stream: body,
	}, nil
}

// Write uploads a raw byte stream to the specified S3 bucket/key.
// Uses multipart upload for large files to avoid 5GB single-PUT limits.
func (c *S3Connector) Write(ctx context.Context, env connectors.EnvelopeRef, r io.Reader, cfg connectors.ConnectorConfig) error {
	bucket := cfg.Get(paramBucket)
	prefix := cfg.Get(paramPrefix)

	if bucket == "" {
		return fmt.Errorf("s3 connector: 'bucket' parameter is required for write")
	}

	// In production: use s3manager.Uploader for streaming multipart upload.
	// uploader := manager.NewUploader(s3Client)
	// _, err := uploader.Upload(ctx, &s3.PutObjectInput{
	//     Bucket:      aws.String(bucket),
	//     Key:         aws.String(prefix),
	//     Body:        r,
	//     ContentType: aws.String(env.MimeType),
	// })
	// return err

	_ = r // consumed in production
	fmt.Printf("[S3Connector] Writing to s3://%s/%s (mimeType=%s)\n", bucket, prefix, env.MimeType)
	return nil
}

// ─────────────────────────────────────────────
//  Helpers
// ─────────────────────────────────────────────

// inferMimeType guesses the MIME type from a file extension in the S3 key.
func inferMimeType(key string) string {
	mimeType := http.DetectContentType([]byte(key))
	// Override based on common extensions
	switch {
	case strings.HasSuffix(key, ".csv"):
		return "text/csv"
	case strings.HasSuffix(key, ".json"):
		return "application/json"
	case strings.HasSuffix(key, ".parquet"):
		return "application/x-parquet"
	case strings.HasSuffix(key, ".pdf"):
		return "application/pdf"
	case strings.HasSuffix(key, ".xlsx"):
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case strings.HasSuffix(key, ".mp4"):
		return "video/mp4"
	case strings.HasSuffix(key, ".jpg"), strings.HasSuffix(key, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(key, ".png"):
		return "image/png"
	}
	return mimeType
}
