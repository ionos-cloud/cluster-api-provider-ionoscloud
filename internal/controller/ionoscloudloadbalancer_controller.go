/*
Copyright 2024 IONOS Cloud.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	"sigs.k8s.io/cluster-api/util/predicates"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/locker"
)

// IonosCloudLoadBalancerReconciler reconciles a IonosCloudLoadBalancer object.
type IonosCloudLoadBalancerReconciler struct {
	client.Client
	scheme *runtime.Scheme
	locker *locker.Locker
}

// NewIonosCloudLoadBalancerReconciler creates a new IonosCloudLoadBalancerReconciler.
func NewIonosCloudLoadBalancerReconciler(mgr ctrl.Manager) *IonosCloudLoadBalancerReconciler {
	r := &IonosCloudLoadBalancerReconciler{
		Client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
		locker: locker.New(),
	}
	return r
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ionoscloudloadbalancers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ionoscloudloadbalancers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ionoscloudloadbalancers/finalizers,verbs=update

//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.4/pkg/reconcile
func (r *IonosCloudLoadBalancerReconciler) Reconcile(
	ctx context.Context,
	ionosCloudLoadBalancer *infrav1.IonosCloudLoadBalancer,
) (ctrl.Result, error) {
	logger := log.FromContext(ctx, "ionoscloudloadbalancer", klog.KObj(ionosCloudLoadBalancer))
	ctx = log.IntoContext(ctx, logger)

	logger.V(4).Info("Reconciling IonosCloudLoadBalancer")

	return ctrl.Result{}, nil
}

func (r *IonosCloudLoadBalancerReconciler) reconcileNormal(
	ctx context.Context,
	ionosCloudLoadBalancer *infrav1.IonosCloudLoadBalancer,
) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func (r *IonosCloudLoadBalancerReconciler) reconcileDelete(
	ctx context.Context,
	ionosCloudLoadBalancer *infrav1.IonosCloudLoadBalancer,
) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IonosCloudLoadBalancerReconciler) SetupWithManager(ctx context.Context,
	mgr ctrl.Manager,
	options controller.Options,
) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(options).
		For(&infrav1.IonosCloudLoadBalancer{}).
		WithEventFilter(predicates.ResourceNotPaused(ctrl.LoggerFrom(ctx))).
		Complete(reconcile.AsReconciler[*infrav1.IonosCloudLoadBalancer](r.Client, r))
}
