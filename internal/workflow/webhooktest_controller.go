package controller

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kttmv1 "github.com/kubeworkflow/flowengine/core/api/v1alpha1"
)

// WebhookTestReconciler reconciles a WebhookTest object
type WebhookTestReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=flowengine.io,resources=webhooktests,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=flowengine.io,resources=webhooktests/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=pods;services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete

func (r *WebhookTestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var wt kttmv1.WebhookTest
	if err := r.Get(ctx, req.NamespacedName, &wt); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// 1. Create Pod if not exists
	podName := fmt.Sprintf("%s-webhook-pod", wt.Name)
	var pod corev1.Pod
	err := r.Get(ctx, client.ObjectKey{Name: podName, Namespace: wt.Namespace}, &pod)
	if apierrors.IsNotFound(err) {
		logger.Info("Creating Pod for WebhookTest", "pod", podName)
		pod = corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      podName,
				Namespace: wt.Namespace,
				Labels:    map[string]string{"app": "webhook-test", "test-id": wt.Name},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "webhook-listener",
						Image: "hashicorp/http-echo", // Mock image for now; replace with actual flowengine server
						Args: []string{
							fmt.Sprintf("-text=Mock Webhook Listener for %s", wt.Name),
							fmt.Sprintf("-listen=:%d", wt.Spec.Port),
						},
						Ports: []corev1.ContainerPort{
							{ContainerPort: wt.Spec.Port},
						},
					},
				},
			},
		}
		if err := ctrl.SetControllerReference(&wt, &pod, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}
		if err := r.Create(ctx, &pod); err != nil {
			return ctrl.Result{}, err
		}
	} else if err != nil {
		return ctrl.Result{}, err
	}

	// 2. Create Service if not exists
	svcName := fmt.Sprintf("%s-webhook-svc", wt.Name)
	var svc corev1.Service
	err = r.Get(ctx, client.ObjectKey{Name: svcName, Namespace: wt.Namespace}, &svc)
	if apierrors.IsNotFound(err) {
		logger.Info("Creating Service for WebhookTest", "service", svcName)
		svc = corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      svcName,
				Namespace: wt.Namespace,
			},
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{"app": "webhook-test", "test-id": wt.Name},
				Ports: []corev1.ServicePort{
					{
						Port:       wt.Spec.Port,
						TargetPort: intstr.FromInt32(wt.Spec.Port),
					},
				},
			},
		}
		if err := ctrl.SetControllerReference(&wt, &svc, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}
		if err := r.Create(ctx, &svc); err != nil {
			return ctrl.Result{}, err
		}
	} else if err != nil {
		return ctrl.Result{}, err
	}

	// 2.5 Create Ingress if Exposure == "External (Public)"
	if wt.Spec.Exposure == "External (Public)" {
		ingName := fmt.Sprintf("%s-webhook-ing", wt.Name)
		var ing networkingv1.Ingress
		err = r.Get(ctx, client.ObjectKey{Name: ingName, Namespace: wt.Namespace}, &ing)
		if apierrors.IsNotFound(err) {
			logger.Info("Creating Ingress for WebhookTest", "ingress", ingName)
			pathType := networkingv1.PathTypePrefix
			ing = networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ingName,
					Namespace: wt.Namespace,
					Annotations: map[string]string{
						"nginx.ingress.kubernetes.io/rewrite-target": "/$2",
					},
				},
				Spec: networkingv1.IngressSpec{
					IngressClassName: func() *string { s := "nginx"; return &s }(),
					Rules: []networkingv1.IngressRule{
						{
							Host: "localhost",
							IngressRuleValue: networkingv1.IngressRuleValue{
								HTTP: &networkingv1.HTTPIngressRuleValue{
									Paths: []networkingv1.HTTPIngressPath{
										{
											Path:     fmt.Sprintf("/test/%s(/|$)(.*)", wt.Name),
											PathType: &pathType,
											Backend: networkingv1.IngressBackend{
												Service: &networkingv1.IngressServiceBackend{
													Name: svcName,
													Port: networkingv1.ServiceBackendPort{
														Number: wt.Spec.Port,
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			}
			refErr := ctrl.SetControllerReference(&wt, &ing, r.Scheme)
			if refErr == nil {
				refErr = r.Create(ctx, &ing)
			}
			if refErr != nil {
				return ctrl.Result{}, refErr
			}
		} else if err != nil {
			return ctrl.Result{}, err
		}
	}

	// 3. Update Status
	var url string
	if wt.Spec.Exposure == "External (Public)" {
		url = fmt.Sprintf("http://localhost:8080/test/%s%s", wt.Name, wt.Spec.Path)
	} else {
		url = fmt.Sprintf("http://%s.%s.svc.cluster.local:%d%s", svcName, wt.Namespace, wt.Spec.Port, wt.Spec.Path)
	}

	if wt.Status.Phase != "Running" || wt.Status.URL != url {
		wt.Status.Phase = "Running"
		wt.Status.URL = url
		if err := r.Status().Update(ctx, &wt); err != nil {
			return ctrl.Result{}, err
		}
	}

	// 4. Garbage Collection / TTL could go here (e.g. check CreationTimestamp, delete if > 10m)
	if time.Since(wt.CreationTimestamp.Time) > 10*time.Minute {
		logger.Info("WebhookTest TTL expired, deleting", "name", wt.Name)
		if err := r.Delete(ctx, &wt); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	return ctrl.Result{RequeueAfter: time.Minute}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WebhookTestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kttmv1.WebhookTest{}).
		Owns(&corev1.Pod{}).
		Owns(&corev1.Service{}).
		Owns(&networkingv1.Ingress{}).
		Complete(r)
}
