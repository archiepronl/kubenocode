package bff

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	authorizationv1 "k8s.io/api/authorization/v1"
)

// HandleGetPermissions checks the user's token and determines their UI permissions
// using Kubernetes SubjectAccessReview or falls back to a default role.
func HandleGetPermissions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")

	// Default mocked fallback permissions if k8s client is not fully configured for auth
	permissions := []string{}

	if token == "mock-jwt-token" {
		// Mock token for demo purposes, granting full access
		permissions = []string{"app:create", "app:read", "app:update", "app:delete", "admin"}
	} else if k8sClient != nil {
		// Example of checking K8s SubjectAccessReview for specific permissions
		// Real implementation would do TokenReview first or use impersonation

		actions := []authorizationv1.ResourceAttributes{
			{Verb: "create", Group: "flowengine.io", Resource: "kttmapps"},
			{Verb: "get", Group: "flowengine.io", Resource: "kttmapps"},
			{Verb: "update", Group: "flowengine.io", Resource: "kttmapps"},
			{Verb: "delete", Group: "flowengine.io", Resource: "kttmapps"},
		}

		for _, action := range actions {
			sar := &authorizationv1.SubjectAccessReview{
				Spec: authorizationv1.SubjectAccessReviewSpec{
					ResourceAttributes: &action,
					User:               "admin", // TODO: Extract from TokenReview
				},
			}

			err := k8sClient.Create(context.Background(), sar)
			if err != nil {
				log.Printf("SAR check failed for %s: %v", action.Verb, err)
				continue
			}

			if sar.Status.Allowed {
				permName := mapVerbToUIPermission(action.Verb)
				if permName != "" {
					permissions = append(permissions, permName)
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"permissions": permissions,
	})
}

func mapVerbToUIPermission(verb string) string {
	switch verb {
	case "create":
		return "app:create"
	case "get", "list":
		return "app:read"
	case "update", "patch":
		return "app:update"
	case "delete":
		return "app:delete"
	default:
		return ""
	}
}
