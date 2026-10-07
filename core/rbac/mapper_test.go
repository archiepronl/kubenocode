package rbac

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	kttmv1 "github.com/kubeworkflow/flowengine/core/api/v1alpha1"
	authv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func TestPermissionMappingAndSets(t *testing.T) {
	tests := []struct {
		permission kttmv1.KttmPermission
		verb       string
		resource   string
	}{
		{kttmv1.PermAppCreate, "create", "kttmapps"},
		{kttmv1.PermAppModify, "update", "kttmapps"},
		{kttmv1.PermAppDelete, "delete", "kttmapps"},
		{kttmv1.PermAppDebug, "get", "kttmapps/debug"},
		{kttmv1.PermAppExport, "get", "kttmapps/export"},
		{kttmv1.PermAppExecute, "create", "kttmapps/trigger"},
		{kttmv1.PermRBACManage, "create", "kttmrolebindings"},
		{kttmv1.PermInfraInstall, "create", "kttminfra"},
		{kttmv1.PermInfraUpgrade, "update", "kttminfra"},
		{kttmv1.PermBundleImport, "create", "kttmbundles"},
		{kttmv1.PermAuditView, "get", "kttmauditlogs"},
		{kttmv1.PermCostView, "get", "kttmcosts"},
		{kttmv1.KttmPermission("unknown"), "get", "kttmapps"},
	}
	for _, tt := range tests {
		verb, resource := kttmPermToK8s(string(tt.permission))
		if verb != tt.verb || resource != tt.resource {
			t.Errorf("kttmPermToK8s(%q) = (%q, %q), want (%q, %q)", tt.permission, verb, resource, tt.verb, tt.resource)
		}
	}

	set := NewPermissionSet([]kttmv1.KttmPermission{kttmv1.PermAppCreate, kttmv1.PermAppModify, kttmv1.PermAppDebug})
	if !set.Has(kttmv1.PermAppCreate) || !set.IsDeveloper() || set.IsAdmin() || set.IsEndUserOnly() {
		t.Fatalf("unexpected developer permission set: %v", set)
	}
	if len(set.ToSlice()) != 3 || NewPermissionSet([]kttmv1.KttmPermission{kttmv1.PermRBACManage}).IsAdmin() != true {
		t.Fatal("permission set conversion or admin check failed")
	}
	if !NewPermissionSet([]kttmv1.KttmPermission{kttmv1.PermAppExecute}).IsEndUserOnly() {
		t.Fatal("end-user-only permission set was not recognized")
	}
	if (PermissionSet{}).Has(kttmv1.PermAppCreate) || (PermissionSet{}).IsDeveloper() || (PermissionSet{}).IsAdmin() || (PermissionSet{}).IsEndUserOnly() {
		t.Fatal("empty permission set should not grant permissions")
	}
}

func TestResolvePermissionsUsesKubernetesAndCustomRoles(t *testing.T) {
	mapper := newTestMapper(t, func(review *authv1.SubjectAccessReview) bool {
		allowed := review.Spec.ResourceAttributes.Resource == "kttmapps" && review.Spec.ResourceAttributes.Verb == "create"
		return allowed
	}, false)
	app := &kttmv1.KttmApp{Spec: kttmv1.KttmAppSpec{RBAC: kttmv1.KttmRBACSpec{Roles: []kttmv1.KttmRole{
		{Name: "data-team", Permissions: []string{"custom:read"}},
		{Name: "alice", Permissions: []string{"custom:write"}},
	}}}}
	principal := Principal{Username: "alice", Groups: []string{"data-team"}}
	permissions, err := mapper.ResolvePermissions(context.Background(), principal, app)
	if err != nil {
		t.Fatalf("ResolvePermissions() error = %v", err)
	}
	set := NewPermissionSet(permissions)
	if !set.Has(kttmv1.PermAppCreate) || !set.Has(kttmv1.KttmPermission("custom:read")) || !set.Has(kttmv1.KttmPermission("custom:write")) || set.Has(kttmv1.PermAppDelete) {
		t.Fatalf("unexpected merged permission set: %v", set)
	}
	allowed, err := mapper.HasPermission(context.Background(), principal, app, kttmv1.PermAppCreate)
	if err != nil || !allowed {
		t.Fatalf("HasPermission(create) = (%t, %v)", allowed, err)
	}
	allowed, err = mapper.HasPermission(context.Background(), principal, app, kttmv1.PermCostView)
	if err != nil || allowed {
		t.Fatalf("HasPermission(cost) = (%t, %v), want false", allowed, err)
	}
}

func TestResolvePermissionsFallbackAndAuthorizationErrors(t *testing.T) {
	mapper := newTestMapper(t, nil, true)
	permissions, err := mapper.ResolvePermissions(context.Background(), Principal{Username: "alice", Groups: []string{"unrecognized"}}, nil)
	if err != nil || len(permissions) != 1 || permissions[0] != kttmv1.PermAppExecute {
		t.Fatalf("default fallback = (%v, %v), want end-user permission", permissions, err)
	}
	app := &kttmv1.KttmApp{Spec: kttmv1.KttmAppSpec{RBAC: kttmv1.KttmRBACSpec{Roles: []kttmv1.KttmRole{{Name: "operator", Permissions: []string{"custom:operate"}}}}}}
	permissions, err = mapper.ResolvePermissions(context.Background(), Principal{Groups: []string{"developer", "operator"}}, app)
	if err != nil {
		t.Fatalf("ResolvePermissions(fallback roles) error = %v", err)
	}
	set := NewPermissionSet(permissions)
	if !set.Has(kttmv1.PermAppCreate) || !set.Has(kttmv1.KttmPermission("custom:operate")) {
		t.Fatalf("fallback role permissions missing: %v", set)
	}
	if _, err := mapper.checkK8sAccess(context.Background(), Principal{Username: "alice"}, string(kttmv1.PermAppCreate)); err == nil {
		t.Fatal("checkK8sAccess() swallowed authorization API error")
	}
}

func TestSubjectAccessReviewCarriesPrincipalAndResource(t *testing.T) {
	mapper := newTestMapper(t, func(review *authv1.SubjectAccessReview) bool {
		if review.Spec.User != "alice" || len(review.Spec.Groups) != 1 || review.Spec.Groups[0] != "operators" ||
			review.Spec.ResourceAttributes.Namespace != "kttm-system" || review.Spec.ResourceAttributes.Group != "kttm.io" ||
			review.Spec.ResourceAttributes.Verb != "get" || review.Spec.ResourceAttributes.Resource != "kttmapps/debug" {
			t.Fatalf("unexpected SAR spec: %+v", review.Spec)
		}
		return true
	}, false)
	allowed, err := mapper.checkK8sAccess(context.Background(), Principal{Username: "alice", Groups: []string{"operators"}}, string(kttmv1.PermAppDebug))
	if err != nil || !allowed {
		t.Fatalf("checkK8sAccess() = (%t, %v)", allowed, err)
	}
}

func TestHasPermissionPropagatesResolverError(t *testing.T) {
	wantErr := errors.New("permission resolution failed")
	mapper := &Mapper{resolvePermissions: func(context.Context, Principal, *kttmv1.KttmApp) ([]kttmv1.KttmPermission, error) {
		return nil, wantErr
	}}
	allowed, err := mapper.HasPermission(context.Background(), Principal{}, nil, kttmv1.PermAppCreate)
	if allowed || !errors.Is(err, wantErr) {
		t.Fatalf("HasPermission() = (%t, %v), want resolver error", allowed, err)
	}
	if mapper.principalHasRole(Principal{Username: "alice", Groups: []string{"team"}}, "missing") {
		t.Fatal("principalHasRole() matched an unrelated role")
	}
}

func newTestMapper(t *testing.T, allowed func(*authv1.SubjectAccessReview) bool, fail bool) *Mapper {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if fail {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(metav1.Status{
				TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Status"},
				Status:   metav1.StatusFailure, Message: "authorization API unavailable", Code: http.StatusInternalServerError,
			})
			return
		}
		var review authv1.SubjectAccessReview
		if err := json.NewDecoder(r.Body).Decode(&review); err != nil {
			t.Errorf("decode SubjectAccessReview: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if allowed != nil {
			review.Status.Allowed = allowed(&review)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(review); err != nil {
			t.Errorf("encode SubjectAccessReview: %v", err)
		}
	}))
	t.Cleanup(server.Close)
	client, err := kubernetes.NewForConfig(&rest.Config{Host: server.URL, QPS: 100, Burst: 100})
	if err != nil {
		t.Fatalf("create Kubernetes client: %v", err)
	}
	return NewMapper(client)
}
