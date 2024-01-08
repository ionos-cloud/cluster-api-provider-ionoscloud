/*
Copyright 2023-2024 IONOS Cloud.

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
	"net/http"

	"github.com/go-logr/logr"
	sdk "github.com/ionos-cloud/sdk-go/v6"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	"k8s.io/utils/pointer"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/ionoscloud"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/pkg/scope"
)

// IonosCloudMachineReconciler reconciles a IonosCloudMachine object.
type IonosCloudMachineReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	IonosCloudClient ionoscloud.Client
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ionoscloudmachines,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ionoscloudmachines/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=ionoscloudmachines/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the IonosCloudMachine object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.0/pkg/reconcile
func (r *IonosCloudMachineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	logger := ctrl.LoggerFrom(ctx)

	ionosCloudMachine := &infrav1.IonosCloudMachine{}
	if err := r.Client.Get(ctx, req.NamespacedName, ionosCloudMachine); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Fetch the Machine.
	machine, err := util.GetOwnerMachine(ctx, r.Client, ionosCloudMachine.ObjectMeta)
	if err != nil {
		return ctrl.Result{}, err
	}
	if machine == nil {
		logger.Info("machine controller has not yet set OwnerRef")
		return ctrl.Result{}, nil
	}

	logger = logger.WithValues("machine", klog.KObj(machine))

	// Fetch the Cluster.
	cluster, err := util.GetClusterFromMetadata(ctx, r.Client, machine.ObjectMeta)
	if err != nil {
		logger.Info("machine is missing cluster label or cluster does not exist")
		return ctrl.Result{}, err
	}

	if annotations.IsPaused(cluster, ionosCloudMachine) {
		logger.Info("ionos cloud machine or linked cluster is marked as paused, not reconciling")
		return ctrl.Result{}, nil
	}

	logger = logger.WithValues("cluster", klog.KObj(cluster))

	infraCluster, err := r.getInfraCluster(ctx, &logger, cluster, ionosCloudMachine)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error getting infra provider cluster or control plane object: %w", err)
	}
	if infraCluster == nil {
		logger.Info("ionos cloud machine is not ready yet")
		return ctrl.Result{}, nil
	}

	// Create the machine scope
	machineScope, err := scope.NewMachineScope(scope.MachineScopeParams{
		Client:            r.Client,
		Cluster:           cluster,
		Machine:           machine,
		InfraCluster:      infraCluster,
		IonosCloudMachine: ionosCloudMachine,
		Logger:            &logger,
	})
	if err != nil {
		logger.Error(err, "failed to create scope")
		return ctrl.Result{}, err
	}

	//// Always close the scope when exiting this function, so we can persist any ProxmoxMachine changes.
	// defer func() {
	//	if err := machineScope.Close(); err != nil && reterr == nil {
	//		reterr = err
	//	}
	// }()

	if !ionosCloudMachine.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, machineScope)
	}

	return r.reconcileNormal(ctx, machineScope)
}

func (r *IonosCloudMachineReconciler) reconcileNormal(
	ctx context.Context, machineScope *scope.MachineScope,
) (ctrl.Result, error) {
	lan, err := r.reconcileLAN(ctx, machineScope)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("could not ensure lan: %w", err)
	}
	if lan == nil {
		return ctrl.Result{Requeue: true}, nil
	}
	return ctrl.Result{}, nil
}

func (r *IonosCloudMachineReconciler) reconcileDelete(
	ctx context.Context, machineScope *scope.MachineScope,
) (ctrl.Result, error) {
	shouldProceed, err := r.reconcileLANDelete(ctx, machineScope)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("could not ensure lan: %w", err)
	}
	if err == nil && !shouldProceed {
		return ctrl.Result{Requeue: true}, nil
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IonosCloudMachineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.IonosCloudMachine{}).
		Watches(
			&clusterv1.Machine{},
			handler.EnqueueRequestsFromMapFunc(util.MachineToInfrastructureMapFunc(infrav1.GroupVersion.WithKind(infrav1.IonosCloudMachineType)))).
		Complete(r)
}

func (r *IonosCloudMachineReconciler) getInfraCluster(
	ctx context.Context, logger *logr.Logger, cluster *clusterv1.Cluster, ionosCloudMachine *infrav1.IonosCloudMachine,
) (*scope.ClusterScope, error) {
	var clusterScope *scope.ClusterScope
	var err error

	ionosCloudCluster := &infrav1.IonosCloudCluster{}

	infraClusterName := client.ObjectKey{
		Namespace: ionosCloudMachine.Namespace,
		Name:      cluster.Spec.InfrastructureRef.Name,
	}

	if err := r.Client.Get(ctx, infraClusterName, ionosCloudCluster); err != nil {
		// IonosCloudCluster is not ready
		return nil, nil //nolint:nilerr
	}

	// Create the cluster scope
	clusterScope, err = scope.NewClusterScope(scope.ClusterScopeParams{
		Client:       r.Client,
		Logger:       logger,
		Cluster:      cluster,
		IonosCluster: ionosCloudCluster,
		IonosClient:  r.IonosCloudClient,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to creat cluster scope: %w", err)
	}

	return clusterScope, nil
}

const lanFormatString = "%s-k8s-lan"

func (r *IonosCloudMachineReconciler) reconcileLAN(
	ctx context.Context, machineScope *scope.MachineScope,
) (*sdk.Lan, error) {
	logger := machineScope.Logger
	dataCenterID := machineScope.IonosCloudMachine.Spec.DatacenterID
	ionos := r.IonosCloudClient
	clusterScope := machineScope.ClusterScope
	clusterName := clusterScope.Cluster.Name
	var err error
	var lan *sdk.Lan

	// try to find available LAN
	lan, err = r.findLANWithinDatacenterLANs(ctx, machineScope)
	if err != nil {
		return nil, fmt.Errorf("could not search for LAN within LAN list: %w", err)
	}
	if lan == nil {
		// check if there is a provisioning request
		reqStatus, err := r.checkProvisioningRequest(ctx, machineScope)
		if err != nil && reqStatus == "" {
			return nil, fmt.Errorf("could not check status of provisioning request: %w", err)
		}
		if reqStatus != "" {
			req := clusterScope.IonosCluster.Status.PendingRequests[dataCenterID]
			l := logger.WithValues(
				"requestURL", req.RequestPath,
				"requestMethod", req.Method,
				"requestStatus", req.State)
			switch reqStatus {
			case string(infrav1.RequestStatusFailed):
				delete(clusterScope.IonosCluster.Status.PendingRequests, dataCenterID)
				return nil, fmt.Errorf("provisioning request has failed: %w", err)
			case string(infrav1.RequestStatusQueued), string(infrav1.RequestStatusRunning):
				l.Info("provisioning request hasn't finished yet. trying again later.")
				return nil, nil
			case string(infrav1.RequestStatusDone):
				lan, err = r.findLANWithinDatacenterLANs(ctx, machineScope)
				if err != nil {
					return nil, fmt.Errorf("could not search for lan within lan list: %w", err)
				}
				if lan == nil {
					l.Info("pending provisioning request has finished, but lan could not be found. trying again later.")
					return nil, nil
				}
			}
		}
	} else {
		return lan, nil
	}
	// request LAN creation
	requestURL, err := ionos.CreateLAN(ctx, dataCenterID, sdk.LanPropertiesPost{
		Name:   pointer.String(fmt.Sprintf(lanFormatString, clusterName)),
		Public: pointer.Bool(true),
	})
	if err != nil {
		return nil, fmt.Errorf("could not create a new LAN: %w ", err)
	}
	clusterScope.IonosCluster.Status.PendingRequests[dataCenterID] = &infrav1.ProvisioningRequest{Method: requestURL}
	logger.WithValues("requestURL", requestURL).Info("new LAN creation was requested")

	return nil, nil
}

func (r *IonosCloudMachineReconciler) findLANWithinDatacenterLANs(
	ctx context.Context, machineScope *scope.MachineScope,
) (lan *sdk.Lan, err error) {
	dataCenterID := machineScope.IonosCloudMachine.Spec.DatacenterID
	ionos := r.IonosCloudClient
	clusterScope := machineScope.ClusterScope
	clusterName := clusterScope.Cluster.Name

	lans, err := ionos.ListLANs(ctx, dataCenterID)
	if err != nil {
		return nil, fmt.Errorf("could not list lans: %w", err)
	}
	if lans.Items != nil {
		for _, lan := range *(lans.Items) {
			if name := lan.Properties.Name; name != nil && *name == fmt.Sprintf(lanFormatString, clusterName) {
				return &lan, nil
			}
		}
	}
	return nil, nil
}

func (r *IonosCloudMachineReconciler) checkProvisioningRequest(
	ctx context.Context, machineScope *scope.MachineScope,
) (string, error) {
	clusterScope := machineScope.ClusterScope
	ionos := r.IonosCloudClient
	dataCenterID := machineScope.IonosCloudMachine.Spec.DatacenterID
	request, requestExists := clusterScope.IonosCluster.Status.PendingRequests[dataCenterID]

	if requestExists {
		reqStatus, err := ionos.CheckRequestStatus(ctx, request.RequestPath)
		if err != nil {
			return "", fmt.Errorf("could not check status of provisioning request: %w", err)
		}
		clusterScope.IonosCluster.Status.PendingRequests[dataCenterID].State = infrav1.RequestStatus(*reqStatus.Metadata.Status)
		clusterScope.IonosCluster.Status.PendingRequests[dataCenterID].Message = *reqStatus.Metadata.Message
		if *reqStatus.Metadata.Status != sdk.RequestStatusDone {
			if metadata := *reqStatus.Metadata; *metadata.Status == sdk.RequestStatusFailed {
				return sdk.RequestStatusFailed, errors.New(*metadata.Message)
			}
			return *reqStatus.Metadata.Status, nil
		}
	}
	return "", nil
}

func (r *IonosCloudMachineReconciler) reconcileLANDelete(ctx context.Context, machineScope *scope.MachineScope) (bool, error) {
	logger := machineScope.Logger
	clusterScope := machineScope.ClusterScope
	dataCenterID := machineScope.IonosCloudMachine.Spec.DatacenterID
	lan, err := r.findLANWithinDatacenterLANs(ctx, machineScope)
	if err != nil {
		return false, fmt.Errorf("error while trying to find lan: %w", err)
	}
	// Check if there is a provisioning request going on
	if lan != nil {
		reqStatus, err := r.checkProvisioningRequest(ctx, machineScope)
		if err != nil && reqStatus == "" {
			return false, fmt.Errorf("could not check status of provisioning request: %w", err)
		}
		if reqStatus != "" {
			req := clusterScope.IonosCluster.Status.PendingRequests[dataCenterID]
			l := logger.WithValues(
				"requestURL", req.RequestPath,
				"requestMethod", req.Method,
				"requestStatus", req.State)
			switch reqStatus {
			case string(infrav1.RequestStatusFailed):
				delete(clusterScope.IonosCluster.Status.PendingRequests, dataCenterID)
				return false, fmt.Errorf("provisioning request has failed: %w", err)
			case string(infrav1.RequestStatusQueued), string(infrav1.RequestStatusRunning):
				l.Info("provisioning request hasn't finished yet. trying again later.")
				return false, nil
			case string(infrav1.RequestStatusDone):
				lan, err = r.findLANWithinDatacenterLANs(ctx, machineScope)
				if err != nil {
					return false, fmt.Errorf("could not search for lan within lan list: %w", err)
				}
				if lan != nil {
					l.Info("pending provisioning request has finished, but lan could still be found. trying again later.")
					return false, nil
				}
			}
		}
	}
	if lan == nil {
		logger.Info("lan seems to be deleted.")
		return true, nil
	}
	if lan.Entities.HasNics() {
		logger.Info("lan seems like it is still being used. let whoever still uses it delete it.")
		// NOTE: the LAN isn't deleted, but we can use the bool to signalize that we can proceed with the machine deletion.
		return true, nil
	}
	requestURL, err := r.IonosCloudClient.DestroyLAN(ctx, dataCenterID, *lan.Id)
	if err != nil {
		return false, fmt.Errorf("could not destroy lan: %w", err)
	}
	machineScope.ClusterScope.IonosCluster.Status.PendingRequests[dataCenterID] = &infrav1.ProvisioningRequest{
		Method:      http.MethodDelete,
		RequestPath: requestURL,
	}
	logger.WithValues("requestURL", requestURL).Info("requested LAN deletion")
	return false, nil
}
