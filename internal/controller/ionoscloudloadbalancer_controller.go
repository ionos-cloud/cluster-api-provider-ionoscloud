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
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/predicates"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/loadbalancing"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/service/cloud"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/util/locker"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/scope"
)

var errOwnerReferenceMissing = errors.New("owner reference not yet applied by cluster controller")

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
	logger := log.FromContext(ctx)

	logger.V(4).Info("Reconciling IonosCloudLoadBalancer")

	ionosCluster, err := r.getIonosCluster(ctx, ionosCloudLoadBalancer.ObjectMeta)
	if err != nil {
		if errors.Is(err, errOwnerReferenceMissing) {
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

	if err = r.validateLoadBalancerSource(ionosCloudLoadBalancer.Spec.LoadBalancerSource); err != nil {
		return ctrl.Result{}, reconcile.TerminalError(err)
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

	cloudService, err := createServiceFromCluster(ctx, r.Client, ionosCluster, logger)
	if err != nil {
		return ctrl.Result{}, err
	}

	prov, err := loadbalancing.NewProvisioner(cloudService, ionosCloudLoadBalancer.Spec.LoadBalancerSource)
	if err != nil {
		return ctrl.Result{}, err
	}

	if !ionosCloudLoadBalancer.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, loadBalancerScope, prov, cloudService)
	}

	return r.reconcileNormal(ctx, loadBalancerScope, prov, cloudService)
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

		return &ionosCluster, nil
	}

	return nil, errOwnerReferenceMissing
}

func (r *IonosCloudLoadBalancerReconciler) reconcileNormal(
	ctx context.Context,
	loadBalancerScope *scope.LoadBalancer,
	prov loadbalancing.Provisioner,
	cloudService *cloud.Service,
) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(4).Info("Reconciling IonosCloudLoadBalancer")

	controllerutil.AddFinalizer(loadBalancerScope.LoadBalancer, infrav1.LoadBalancerFinalizer)

	if err := r.validateEndpoints(loadBalancerScope); err != nil {
		conditions.MarkFalse(
			loadBalancerScope.LoadBalancer,
			infrav1.LoadBalancerReadyCondition,
			infrav1.InvalidEndpointConfigurationReason,
			clusterv1.ConditionSeverityError, "")
		return ctrl.Result{}, reconcile.TerminalError(err)
	}

	if requeue, err := r.checkRequestStatus(ctx, loadBalancerScope, cloudService); err != nil || requeue {
		return ctrl.Result{Requeue: requeue}, err
	}

	if requeue, err := prov.Provision(ctx, loadBalancerScope); err != nil || requeue {
		if err != nil {
			err = fmt.Errorf("error during provisioning: %w", err)
		}

		return ctrl.Result{RequeueAfter: defaultReconcileDuration}, err
	}

	conditions.MarkTrue(loadBalancerScope.LoadBalancer, infrav1.LoadBalancerReadyCondition)
	loadBalancerScope.LoadBalancer.Status.Ready = true

	logger.V(4).Info("Successfully reconciled IonosCloudLoadBalancer")
	return ctrl.Result{}, nil
}

func (r *IonosCloudLoadBalancerReconciler) reconcileDelete(
	ctx context.Context,
	loadBalancerScope *scope.LoadBalancer,
	prov loadbalancing.Provisioner,
	cloudService *cloud.Service,
) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(4).Info("Deleting IonosCloudLoadBalancer")

	if requeue, err := r.checkRequestStatus(ctx, loadBalancerScope, cloudService); err != nil || requeue {
		return ctrl.Result{Requeue: requeue}, err
	}

	if requeue, err := prov.Destroy(ctx, loadBalancerScope); err != nil || requeue {
		if err != nil {
			err = fmt.Errorf("error during cleanup: %w", err)
		}

		return ctrl.Result{RequeueAfter: defaultReconcileDuration}, err
	}

	controllerutil.RemoveFinalizer(loadBalancerScope.LoadBalancer, infrav1.LoadBalancerFinalizer)
	logger.V(4).Info("Successfully deleted IonosCloudLoadBalancer")
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
		Watches(&infrav1.IonosCloudMachine{}, handler.EnqueueRequestsFromMapFunc(r.machineToLoadBalancerRequests)).
		WithEventFilter(predicates.ResourceNotPaused(ctrl.LoggerFrom(ctx))).
		Complete(reconcile.AsReconciler[*infrav1.IonosCloudLoadBalancer](r.Client, r))
}

func (*IonosCloudLoadBalancerReconciler) validateEndpoints(loadBalancerScope *scope.LoadBalancer) error {
	s := loadBalancerScope

	if s.InfraClusterEndpoint().IsValid() && s.Endpoint().IsZero() {
		return errors.New("infra cluster already has an endpoint set, but the load balancer does not")
	}

	if s.InfraClusterEndpoint().IsValid() && s.Endpoint().IsValid() {
		if s.InfraClusterEndpoint() == s.Endpoint() {
			return nil
		}

		return errors.New("infra cluster and load balancer endpoints do not match")
	}

	return nil
}

func (*IonosCloudLoadBalancerReconciler) checkRequestStatus(
	ctx context.Context,
	loadBalancerScope *scope.LoadBalancer,
	cloudService *cloud.Service,
) (requeue bool, retErr error) {
	logger := ctrl.LoggerFrom(ctx)
	loadBalancer := loadBalancerScope.LoadBalancer

	if req := loadBalancer.Status.CurrentRequest; req != nil {
		logger.Info("Checking request status", "request", req.RequestPath, "method", req.Method)
		status, message, err := cloudService.GetRequestStatus(ctx, req.RequestPath)
		if err != nil {
			retErr = fmt.Errorf("could not get request status: %w", err)
		} else {
			requeue, retErr = withStatus(status, message, &logger, func() error {
				loadBalancer.DeleteCurrentRequest()
				return nil
			})
		}
	}

	return requeue, retErr
}

func (*IonosCloudLoadBalancerReconciler) validateLoadBalancerSource(source infrav1.LoadBalancerSource) error {
	if source.NLB == nil {
		return errors.New("exactly one source needs to be set, NLB is not set")
	}

	return nil
}

func (r *IonosCloudLoadBalancerReconciler) machineToLoadBalancerRequests(ctx context.Context, obj client.Object) []reconcile.Request {
	logger := ctrl.LoggerFrom(ctx)
	machine, ok := obj.(*infrav1.IonosCloudMachine)
	if !ok {
		return nil
	}

	if !isControlPlaneMachine(machine) || !isMachineAvailable(machine) {
		return nil
	}

	infraCluster, err := getInfraClusterFromMachine(ctx, r.Client, machine)
	if err != nil {
		logger.Error(err, "failed to get infra cluster from machine")
		return nil
	}

	if infraCluster.Spec.LoadBalancerProviderRef == nil {
		logger.Info("Load balancer provider ref is not set, skipping load balancer reconciliation")
		return nil
	}

	return []reconcile.Request{{
		NamespacedName: client.ObjectKey{
			Namespace: infraCluster.GetNamespace(),
			Name:      infraCluster.Spec.LoadBalancerProviderRef.Name,
		},
	}}
}

func isControlPlaneMachine(machine *infrav1.IonosCloudMachine) bool {
	labels := machine.GetLabels()
	if labels == nil {
		return false
	}

	_, ok := labels[clusterv1.MachineControlPlaneLabel]
	return ok
}

func isMachineAvailable(machine *infrav1.IonosCloudMachine) bool {
	return machine.DeletionTimestamp.IsZero() && conditions.IsTrue(machine, clusterv1.ReadyCondition)
}

func getInfraClusterFromMachine(ctx context.Context, c client.Client, machine *infrav1.IonosCloudMachine) (*infrav1.IonosCloudCluster, error) {
	labels := machine.GetLabels()
	if labels == nil {
		return nil, errors.New("machine has no labels")
	}

	capiCluster, err := util.GetClusterFromMetadata(ctx, c, machine.ObjectMeta)
	if err != nil {
		return nil, err
	}

	if !capiCluster.DeletionTimestamp.IsZero() {
		return nil, errors.New("cluster is already being deleted")
	}

	infraRef := capiCluster.Spec.InfrastructureRef

	infraClusterKey := client.ObjectKey{Namespace: infraRef.Namespace, Name: infraRef.Name}
	var infraCluster infrav1.IonosCloudCluster
	if err := c.Get(ctx, infraClusterKey, &infraCluster); err != nil {
		return nil, err
	}

	return &infraCluster, nil
}
