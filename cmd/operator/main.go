// Package main is the entrypoint for the FlowEngine Kubernetes Operator.
//
// It bootstraps the controller-runtime manager, registers the FullStackApplication
// CRD scheme, and starts the reconcile loop. The manager handles leader election,
// health probes, and graceful shutdown.
//
// Resource footprint targets (NFR-1):
//   - Idle memory: < 100MB
//   - CPU: < 10m idle
//
// Run locally:
//
//	go run ./cmd/operator/ --kubeconfig=$HOME/.kube/config
package main

import (
	"flag"
	"os"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	v1alpha1 "github.com/kubeworkflow/flowengine/core/api/v1alpha1"
	"github.com/kubeworkflow/flowengine/core/engine"
	"github.com/kubeworkflow/flowengine/core/linter"
	workflow "github.com/kubeworkflow/flowengine/internal/workflow"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
}

func main() {
	var (
		metricsAddr          string
		probeAddr            string
		enableLeaderElection bool
		executionBackend     string
	)

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metrics endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the health probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false, "Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&executionBackend, "execution-backend", "argo", "Execution backend to use (argo|tekton).")
	flag.Parse()

	opts := zap.Options{Development: true}
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: metricsAddr,
		},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "flowengine-operator-leader",
	})
	if err != nil {
		setupLog.Error(err, "Unable to create manager")
		os.Exit(1)
	}

	// Select the execution backend based on the flag
	var compiler engine.Backend
	switch executionBackend {
	case "argo":
		compiler = engine.NewArgoCompiler()
	default:
		setupLog.Error(nil, "Unknown execution backend", "backend", executionBackend)
		os.Exit(1)
	}

	// Wire up the reconciler
	if err := (&workflow.FullStackApplicationReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Linter:   linter.New(),
		Compiler: compiler,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "Unable to create FullStackApplication controller")
		os.Exit(1)
	}

	if err := (&workflow.WebhookTestReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "Unable to create WebhookTest controller")
		os.Exit(1)
	}

	// Health probes — required for Kubernetes liveness/readiness checks
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "Unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "Unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("Starting FlowEngine Operator",
		"backend", executionBackend,
		"metricsAddr", metricsAddr,
		"probeAddr", probeAddr,
		"leaderElection", enableLeaderElection,
	)

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "Problem running manager")
		os.Exit(1)
	}
}
