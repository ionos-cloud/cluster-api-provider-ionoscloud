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
	"errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/predicates"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/loadbalancing"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/locker"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/scope"
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
) (_ ctrl.Result, retErr error) {
	logger := log.FromContext(ctx, "ionoscloudloadbalancer", klog.KObj(ionosCloudLoadBalancer))
	ctx = log.IntoContext(ctx, logger)

	logger.V(4).Info("Reconciling IonosCloudLoadBalancer")

	ionosCluster, err := r.getIonosCluster(ctx, ionosCloudLoadBalancer.ObjectMeta)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("Owner reference not yet applied to IonosCloudLoadBalancer")
			return ctrl.Result{RequeueAfter: defaultReconcileDuration}, nil
		}
		return ctrl.Result{}, err
	}

	cluster, err := util.GetOwnerCluster(ctx, r.Client, ionosCluster.ObjectMeta)
	if err != nil {
		return ctrl.Result{}, err
	}

	if cluster == nil {
		logger.Info("Waiting for cluster controller to set OwnerRef on IonosCloudCluster")
		return ctrl.Result{RequeueAfter: defaultReconcileDuration}, nil
	}

	if annotations.IsPaused(cluster, ionosCloudLoadBalancer) {
		logger.Info("IONOS Cloud load balancer or linked cluster is marked as paused. not reconciling")
		return ctrl.Result{}, nil
	}

	clusterScope, err := scope.NewCluster(scope.ClusterParams{
		Client:       r.Client,
		Cluster:      cluster,
		IonosCluster: ionosCluster,
		Locker:       r.locker,
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	loadBalancerScope, err := scope.NewLoadBalancer(scope.LoadBalancerParams{
		Client:       r.Client,
		LoadBalancer: ionosCloudLoadBalancer,
		ClusterScope: clusterScope,
		Locker:       r.locker,
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	defer func() {
		err := loadBalancerScope.Finalize()
		retErr = errors.Join(retErr, err)
	}()

	prov, err := loadbalancing.NewProvisioner(nil, ionosCloudLoadBalancer.Spec.Type)
	if err != nil {
		return ctrl.Result{}, err
	}

	if !ionosCloudLoadBalancer.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, loadBalancerScope, prov)
	}

	return r.reconcileNormal(ctx, loadBalancerScope, prov)
}

func (r *IonosCloudLoadBalancerReconciler) getIonosCluster(
	ctx context.Context,
	meta metav1.ObjectMeta,
) (*infrav1.IonosCloudCluster, error) {
	var ionosCluster infrav1.IonosCloudCluster
	for _, ref := range meta.GetOwnerReferences() {
		if ref.Kind != infrav1.IonosCloudClusterKind {
			continue
		}

		clusterKey := client.ObjectKey{Namespace: meta.Namespace, Name: ref.Name}
		if err := r.Client.Get(ctx, clusterKey, &ionosCluster); err != nil {
			return nil, err
		}
	}
	return &ionosCluster, nil
}

func (*IonosCloudLoadBalancerReconciler) reconcileNormal(
	ctx context.Context,
	_ *scope.LoadBalancer,
	_ loadbalancing.Provisioner,
) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.V(4).Info("Reconciling IonosCloudLoadBalancer")

	// TODO(lubedacht): Implement reconcileNormal
	//		- Check if the Endpoint was already set in the IonosCloudCluster - Do nothing if it is already set
	// 		- Create Webhook for HA to apply kube-vip configuration to cloud init
	// 		- Update IonosCloudCluster with Provisioner Endpoint
	// 		- Set IonosCloudLoadBalancer status Ready

	return ctrl.Result{}, nil
}

func (*IonosCloudLoadBalancerReconciler) reconcileDelete(
	_ context.Context,
	_ *scope.LoadBalancer,
	_ loadbalancing.Provisioner,
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
