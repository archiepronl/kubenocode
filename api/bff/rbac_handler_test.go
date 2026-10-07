package bff

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleListUsers(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		expectedStatus int
	}{
		{
			name:           "Method Not Allowed",
			method:         http.MethodPost,
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "Success",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, "/api/auth/users", nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(HandleListUsers)
			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
			}

			if tt.expectedStatus == http.StatusOK {
				var users []UserPermissions
				if err := json.Unmarshal(rr.Body.Bytes(), &users); err != nil {
					t.Fatalf("Failed to parse response body: %v", err)
				}
				if len(users) != 3 { // Expecting 3 mock users
					t.Errorf("expected 3 users, got %d", len(users))
				}
			}
		})
	}
}

func TestHandleUpdateUserPermissions(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		username       string
		body           any
		expectedStatus int
	}{
		{
			name:           "Method Not Allowed",
			method:         http.MethodPost,
			username:       "admin@example.com",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "Missing Username",
			method:         http.MethodPut,
			username:       "",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid Body",
			method:         http.MethodPut,
			username:       "admin@example.com",
			body:           "invalid-json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Success",
			method:         http.MethodPut,
			username:       "admin@example.com",
			body:           UpdatePermissionsRequest{Permissions: []string{"app:create"}},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var reqBody []byte
			if tt.body != nil {
				if s, ok := tt.body.(string); ok {
					reqBody = []byte(s)
				} else {
					reqBody, _ = json.Marshal(tt.body)
				}
			}

			req, err := http.NewRequest(tt.method, "/api/auth/users/"+tt.username+"/permissions", bytes.NewBuffer(reqBody))
			if err != nil {
				t.Fatal(err)
			}
			req.SetPathValue("username", tt.username)

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(HandleUpdateUserPermissions)
			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
			}
		})
	}
}
