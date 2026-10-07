package adapters_test

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/kubeworkflow/flowengine/core/engine/adapters"
)

func TestBuiltInAdaptersPreserveEnvelopeAndStream(t *testing.T) {
	tests := []struct {
		name    string
		adapter interface {
			MimeTypes() []string
			Process(context.Context, interface{}, io.Reader) (interface{}, io.ReadCloser, error)
		}
		mimeTypes []string
	}{
		{name: "pdf", adapter: adapters.NewPDFAdapter(), mimeTypes: []string{"application/pdf"}},
		{name: "video", adapter: adapters.NewVideoAdapter(), mimeTypes: []string{"video/"}},
		{name: "image", adapter: adapters.NewImageAdapter(), mimeTypes: []string{"image/"}},
		{name: "audio", adapter: adapters.NewAudioAdapter(), mimeTypes: []string{"audio/"}},
		{name: "excel", adapter: adapters.NewExcelAdapter(), mimeTypes: []string{"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "application/vnd.ms-excel"}},
		{name: "arrow", adapter: adapters.NewArrowAdapter(), mimeTypes: []string{"application/x-arrow-ipc-stream", "application/x-arrow", "application/x-parquet", "application/arrow"}},
		{name: "passthrough", adapter: adapters.NewPassthroughAdapter(), mimeTypes: []string{""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.adapter.MimeTypes(); !equalStrings(got, tt.mimeTypes) {
				t.Fatalf("MimeTypes() = %v, want %v", got, tt.mimeTypes)
			}

			env := &struct{ ID string }{ID: tt.name}
			processedEnv, output, err := tt.adapter.Process(context.Background(), env, strings.NewReader("raw payload"))
			if err != nil {
				t.Fatalf("Process() error = %v", err)
			}
			if processedEnv != env {
				t.Errorf("Process() envelope = %v, want original %v", processedEnv, env)
			}
			if output == nil {
				t.Fatal("Process() returned nil stream")
			}
			data, err := io.ReadAll(output)
			if err != nil {
				t.Fatalf("reading output: %v", err)
			}
			if err := output.Close(); err != nil {
				t.Errorf("closing output: %v", err)
			}
			if string(data) != "raw payload" {
				t.Errorf("output = %q, want original payload", data)
			}
		})
	}
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
