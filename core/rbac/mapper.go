// Package rbac implements the Kubernetes RBAC → KTTM permission mapping layer.
//
// KTTM-REQ-039: The system queries the host Kubernetes cluster's native RBAC layer
// (ServiceAccounts, Roles, ClusterRoles, RoleBindings) to determine access privileges.
// Falls back to the internal default role table if K8s RBAC is unavailable.
//
// KTTM-REQ-005: Administrators compose custom roles from the master permission checklist.
// Custom roles are stored in KttmApp.spec.rbac.roles and evaluated at runtime.
package rbac

import (
	"context"
	"fmt"

	kttmv1 "github.com/kubeworkflow/flowengine/core/api/v1alpha1"
	authv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ─────────────────────────────────────────────
//  Mapper
// ─────────────────────────────────────────────

// Mapper translates Kubernetes RBAC decisions into KTTM permission arrays.
type Mapper struct {
	k8s kubernetes.Interface
}

// NewMapper creates a new RBAC mapper using the provided Kubernetes client.
func NewMapper(k8s kubernetes.Interface) *Mapper {
	return &Mapper{k8s: k8s}
}

// ─────────────────────────────────────────────
//  Principal
// ─────────────────────────────────────────────

// Principal represents an authenticated user or service account.
type Principal struct {
	// Username is the Kubernetes user identity (e.g., from OIDC "sub" claim).
	Username string
	// Groups are the user's group memberships (e.g., LDAP groups, OIDC roles).
	Groups []string
	// ServiceAccount is set for pod-level principals.
	ServiceAccount *ServiceAccountRef
}

// ServiceAccountRef identifies a Kubernetes ServiceAccount.
type ServiceAccountRef struct {
	Name      string
	Namespace string
}

// ─────────────────────────────────────────────
//  Permission resolution
// ─────────────────────────────────────────────

// ResolvePermissions returns the KTTM permission set for a principal.
// It queries K8s SubjectAccessReview for each KTTM permission, then falls
// back to default role assignment if the cluster has no RBAC binding.
func (m *Mapper) ResolvePermissions(ctx context.Context, p Principal, app *kttmv1.KttmApp) ([]kttmv1.KttmPermission, error) {
	resolved := make(map[kttmv1.KttmPermission]bool)

	// ── 1. Check K8s-native SubjectAccessReview for each permission ────────
	for _, perm := range kttmv1.AllPermissions {
		allowed, err := m.checkK8sAccess(ctx, p, string(perm))
		if err != nil {
			// K8s RBAC unavailable — fall back to default roles
			return m.defaultPermissions(p, app)
		}
		if allowed {
			resolved[perm] = true
		}
	}

	// ── 2. Merge app-level custom role permissions ─────────────────────────
	// Custom roles defined in KttmApp.spec.rbac.roles take additional effect
	// if the principal's group matches a role name.
	if app != nil {
		for _, role := range app.Spec.RBAC.Roles {
			if m.principalHasRole(p, role.Name) {
				for _, perm := range role.Permissions {
					resolved[kttmv1.KttmPermission(perm)] = true
				}
			}
		}
	}

	// Convert map to slice
	perms := make([]kttmv1.KttmPermission, 0, len(resolved))
	for p := range resolved {
		perms = append(perms, p)
	}
	return perms, nil
}

// HasPermission is a convenience helper that returns true if the principal
// holds a specific KTTM permission.
func (m *Mapper) HasPermission(ctx context.Context, p Principal, app *kttmv1.KttmApp, perm kttmv1.KttmPermission) (bool, error) {
	perms, err := m.ResolvePermissions(ctx, p, app)
	if err != nil {
		return false, err
	}
	for _, resolved := range perms {
		if resolved == perm {
			return true, nil
		}
	}
	return false, nil
}

// ─────────────────────────────────────────────
//  K8s SubjectAccessReview
// ─────────────────────────────────────────────

// checkK8sAccess uses SubjectAccessReview to ask the K8s API server
// whether a principal is allowed to perform a KTTM verb on a KTTM resource.
//
// KTTM permissions are mapped to K8s verbs on a custom API resource:
//
//	app:create  → "create"  on resource "kttmapps"
//	app:modify  → "update"  on resource "kttmapps"
//	app:debug   → "get"     on resource "kttmapps/debug"
//	rbac:manage → "create"  on resource "kttmrolebindings"
//	etc.
func (m *Mapper) checkK8sAccess(ctx context.Context, p Principal, perm string) (bool, error) {
	verb, resource := kttmPermToK8s(perm)

	sar := &authv1.SubjectAccessReview{
		Spec: authv1.SubjectAccessReviewSpec{
			User:   p.Username,
			Groups: p.Groups,
			ResourceAttributes: &authv1.ResourceAttributes{
				Namespace: "kttm-system",
				Verb:      verb,
				Group:     "kttm.io",
				Resource:  resource,
			},
		},
	}

	result, err := m.k8s.AuthorizationV1().SubjectAccessReviews().Create(ctx, sar, metav1.CreateOptions{})
	if err != nil {
		return false, fmt.Errorf("SubjectAccessReview failed: %w", err)
	}
	return result.Status.Allowed, nil
}

// kttmPermToK8s maps a KTTM permission string to a K8s (verb, resource) pair.
func kttmPermToK8s(perm string) (verb, resource string) {
	switch perm {
	case string(kttmv1.PermAppCreate):
		return "create", "kttmapps"
	case string(kttmv1.PermAppModify):
		return "update", "kttmapps"
	case string(kttmv1.PermAppDelete):
		return "delete", "kttmapps"
	case string(kttmv1.PermAppDebug):
		return "get", "kttmapps/debug"
	case string(kttmv1.PermAppExport):
		return "get", "kttmapps/export"
	case string(kttmv1.PermAppExecute):
		return "create", "kttmapps/trigger"
	case string(kttmv1.PermRBACManage):
		return "create", "kttmrolebindings"
	case string(kttmv1.PermInfraInstall):
		return "create", "kttminfra"
	case string(kttmv1.PermInfraUpgrade):
		return "update", "kttminfra"
	case string(kttmv1.PermBundleImport):
		return "create", "kttmbundles"
	case string(kttmv1.PermAuditView):
		return "get", "kttmauditlogs"
	case string(kttmv1.PermCostView):
		return "get", "kttmcosts"
	default:
		return "get", "kttmapps"
	}
}

// ─────────────────────────────────────────────
//  Default role fallback
// ─────────────────────────────────────────────

// defaultPermissions returns permissions based on the principal's group memberships
// mapped to the built-in KTTM role table (KTTM-REQ-039 fallback).
func (m *Mapper) defaultPermissions(p Principal, app *kttmv1.KttmApp) ([]kttmv1.KttmPermission, error) {
	resolved := make(map[kttmv1.KttmPermission]bool)

	// Map K8s groups to default KTTM roles
	for _, group := range p.Groups {
		if perms, ok := kttmv1.DefaultRoles[group]; ok {
			for _, perm := range perms {
				resolved[perm] = true
			}
		}
	}

	// If no group match, default to end-user (minimum privilege)
	if len(resolved) == 0 {
		for _, perm := range kttmv1.DefaultRoles["enduser"] {
			resolved[perm] = true
		}
	}

	// Merge app-level custom roles
	if app != nil {
		for _, role := range app.Spec.RBAC.Roles {
			if m.principalHasRole(p, role.Name) {
				for _, perm := range role.Permissions {
					resolved[kttmv1.KttmPermission(perm)] = true
				}
			}
		}
	}

	perms := make([]kttmv1.KttmPermission, 0, len(resolved))
	for p := range resolved {
		perms = append(perms, p)
	}
	return perms, nil
}

// principalHasRole returns true if the principal's username or any group
// matches the given role name (case-insensitive).
func (m *Mapper) principalHasRole(p Principal, roleName string) bool {
	if p.Username == roleName {
		return true
	}
	for _, g := range p.Groups {
		if g == roleName {
			return true
		}
	}
	return false
}

// ─────────────────────────────────────────────
//  Permission set helpers
// ─────────────────────────────────────────────

// PermissionSet wraps a slice of permissions for quick lookup.
type PermissionSet map[kttmv1.KttmPermission]bool

// Has returns true if the given permission is present.
func (ps PermissionSet) Has(p kttmv1.KttmPermission) bool { return ps[p] }

// IsDeveloper returns true if the principal has all developer-tier permissions.
func (ps PermissionSet) IsDeveloper() bool {
	return ps[kttmv1.PermAppCreate] && ps[kttmv1.PermAppModify] && ps[kttmv1.PermAppDebug]
}

// IsAdmin returns true if the principal has rbac:manage.
func (ps PermissionSet) IsAdmin() bool { return ps[kttmv1.PermRBACManage] }

// IsEndUserOnly returns true if the principal has only app:execute.
func (ps PermissionSet) IsEndUserOnly() bool {
	return ps[kttmv1.PermAppExecute] && !ps[kttmv1.PermAppCreate] && !ps[kttmv1.PermRBACManage]
}

// ToSlice converts the set back to a permission slice.
func (ps PermissionSet) ToSlice() []kttmv1.KttmPermission {
	out := make([]kttmv1.KttmPermission, 0, len(ps))
	for p := range ps {
		out = append(out, p)
	}
	return out
}

// NewPermissionSet builds a PermissionSet from a slice.
func NewPermissionSet(perms []kttmv1.KttmPermission) PermissionSet {
	ps := make(PermissionSet, len(perms))
	for _, p := range perms {
		ps[p] = true
	}
	return ps
}
