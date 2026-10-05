package controller

import (
	"net/http"
	"testing"

	"github.com/go-logr/logr"
	v1alpha1 "github.com/kubeworkflow/flowengine/core/api/v1alpha1"
	"github.com/kubeworkflow/flowengine/core/engine"
	"github.com/kubeworkflow/flowengine/core/linter"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type setupManager struct {
	manager.Manager
	scheme      *runtime.Scheme
	controllers int
}

func (m *setupManager) Add(manager.Runnable) error {
	m.controllers++
	return nil
}

func (*setupManager) GetControllerOptions() config.Controller { return config.Controller{} }

func (*setupManager) GetCache() cache.Cache { return nil }

func (m *setupManager) GetScheme() *runtime.Scheme { return m.scheme }

func (*setupManager) GetLogger() logr.Logger { return logr.Discard() }

func (*setupManager) AddMetricsServerExtraHandler(string, http.Handler) error { return nil }

func TestSetupWithManagerRegistersController(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := v1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("register API scheme: %v", err)
	}
	mgr := &setupManager{scheme: scheme}
	reconciler := &FullStackApplicationReconciler{Linter: linter.New(), Compiler: engine.NewArgoCompiler()}
	if err := reconciler.SetupWithManager(mgr); err != nil {
		t.Fatalf("SetupWithManager() error = %v", err)
	}
	if mgr.controllers != 1 {
		t.Fatalf("manager received %d controllers, want 1", mgr.controllers)
	}
}