package bff

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleGetPermissions(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		authHeader     string
		expectedStatus int
		expectedPerms  []string
	}{
		{
			name:           "Method Not Allowed",
			method:         http.MethodPost,
			authHeader:     "Bearer mock-jwt-token",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedPerms:  nil,
		},
		{
			name:           "Unauthorized - No Header",
			method:         http.MethodGet,
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			expectedPerms:  nil,
		},
		{
			name:           "Unauthorized - Invalid Format",
			method:         http.MethodGet,
			authHeader:     "mock-jwt-token",
			expectedStatus: http.StatusUnauthorized,
			expectedPerms:  nil,
		},
		{
			name:           "Success - Mock Token",
			method:         http.MethodGet,
			authHeader:     "Bearer mock-jwt-token",
			expectedStatus: http.StatusOK,
			expectedPerms:  []string{"app:create", "app:read", "app:update", "app:delete", "admin"},
		},
		{
			name:           "Success - Empty Permissions for invalid mock",
			method:         http.MethodGet,
			authHeader:     "Bearer invalid-token",
			expectedStatus: http.StatusOK,
			expectedPerms:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, "/api/auth/permissions", nil)
			if err != nil {
				t.Fatal(err)
			}
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(HandleGetPermissions)
			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
			}

			if tt.expectedStatus == http.StatusOK {
				var resp map[string][]string
				if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
					t.Fatalf("Failed to parse response body: %v", err)
				}

				perms := resp["permissions"]
				if len(perms) != len(tt.expectedPerms) {
					t.Errorf("expected %d permissions, got %d", len(tt.expectedPerms), len(perms))
				}

				// Optional: Deep compare logic here if needed
			}
		})
	}
}

func TestMapVerbToUIPermission(t *testing.T) {
	tests := map[string]string{
		"create":  "app:create",
		"get":     "app:read",
		"list":    "app:read",
		"update":  "app:update",
		"patch":   "app:update",
		"delete":  "app:delete",
		"unknown": "",
	}

	for verb, expected := range tests {
		result := mapVerbToUIPermission(verb)
		if result != expected {
			t.Errorf("mapVerbToUIPermission(%s) = %s, want %s", verb, result, expected)
		}
	}
}
