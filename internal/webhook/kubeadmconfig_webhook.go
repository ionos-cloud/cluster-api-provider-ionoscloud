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

// Package webhook contains the webhooks for the IONOS Cloud provider.
package webhook

import (
	"context"
	"fmt"
	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/internal/loadbalancing"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/bootstrap/kubeadm/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// +kubebuilder:webhook:path=/mutate-bootstrap-cluster-x-k8s-io-v1beta1-kubeadmconfig,mutating=true,failurePolicy=fail,sideEffects=None,groups=bootstrap.cluster.x-k8s.io,resources=kubeadmconfigs,verbs=create;update,versions=v1beta1,name=mutation.kubeadmconfig.bootstrap.cluster.x-k8s.io,admissionReviewVersions=v1

// KubeadmConfigWebhook will handle mutating KubeadmConfig objects.
type KubeadmConfigWebhook struct {
	Client client.Client
}

var _ webhook.CustomDefaulter = &KubeadmConfigWebhook{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the type.
func (r *KubeadmConfigWebhook) Default(ctx context.Context, obj runtime.Object) error {
	var allErrs field.ErrorList
	logger := ctrl.LoggerFrom(ctx)

	config, ok := obj.(*v1beta1.KubeadmConfig)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a KubeadmConfig but got a %T", obj))
	}

	// check if label cluster.x-k8s.io/control-plane is set, which indicates that the KubeadmConfig is a control plane config
	configLabels := config.GetLabels()
	if _, exists := configLabels[clusterv1.MachineControlPlaneLabel]; !exists {

		logger.Info("config is not related to control plane. Skipping mutation", "object", obj)
		return nil
	}

	clusterName, exists := configLabels[clusterv1.ClusterNameLabel]
	if !exists {
		allErrs = append(allErrs,
			field.Required(
				field.NewPath(
					"metadata", "labels",
				), clusterv1.ClusterNameLabel+" label is required"))
		//logger.Info("cluster name label not set. Unable to determine parent cluster", "object", obj)
		return apierrors.NewInvalid(v1beta1.GroupVersion.WithKind("KubeadmConfig").GroupKind(), config.Name, allErrs)
	}

	ionosCluster, err := r.queryCluster(ctx, config, clusterName)
	if err != nil {
		return err
	}

	if ionosCluster.Spec.LoadBalancerProviderRef == nil {
		logger.V(4).Info("no load balancer provider ref found in the cluster", "cluster", ionosCluster.Name)
		// nothing to do here
		return nil
	}

	loadBalancer, err := r.queryLoadBalancer(ctx, ionosCluster)
	if err != nil {
		return err
	}

	if loadBalancer.Spec.KubeVIP == nil {
		// mutation is only required for kube-vip
		// nothing to be done here.
		return nil
	}

	if err := r.mutateKubeadmConfig(ctx, config, loadBalancer); err != nil {
		return apierrors.NewInternalError(err)
	}

	// if kube-vip is set, make sure that the file in the kubeadm config contain the kube-vip static pod,
	// add the init script, and the post kubeadm command to replace the super-admin config

	logger.Info("mutating KubeadmConfig", "object", obj)

	if len(allErrs) > 0 {
		return apierrors.NewInvalid(v1beta1.GroupVersion.WithKind("KubeadmConfig").GroupKind(), config.Name, allErrs)
	}

	return nil
}

func (r *KubeadmConfigWebhook) mutateKubeadmConfig(ctx context.Context, config *v1beta1.KubeadmConfig, loadBalancer *infrav1.IonosCloudLoadBalancer) error {
	logger := ctrl.LoggerFrom(ctx)

	logger.Info("mutating KubeadmConfig", "config", config.Name, "load balancer", loadBalancer.Name)

	const (
		kubeVIPManifestPath   = "/etc/kubernetes/manifests/kube-vip.yaml"
		kubeVIPInitScriptPath = "/etc/kube-vip-prepare.sh"
	)

	s := sets.Set[string]{}

	for _, file := range config.Spec.Files {
		s.Insert(file.Path)
	}

	endpoint := loadBalancer.Spec.LoadBalancerEndpoint
	kubeVIPConfig := loadBalancer.Spec.KubeVIP

	if !s.Has(kubeVIPManifestPath) {
		render, err := loadbalancing.RenderKubeVIPConfig(
			endpoint.Host,
			int(endpoint.Port),
			kubeVIPConfig.Image,
			kubeVIPConfig.Interface,
		)

		if err != nil {
			logger.Error(err, "unable to render kube-vip config")
			return err
		}

		config.Spec.Files = append(config.Spec.Files, v1beta1.File{
			Path:    kubeVIPManifestPath,
			Content: render,
			Owner:   "root:root",
		})
	}

	if !s.Has(kubeVIPInitScriptPath) {
		config.Spec.Files = append(config.Spec.Files, v1beta1.File{
			Path:        kubeVIPInitScriptPath,
			Content:     loadbalancing.KubeVIPScript(),
			Permissions: "0700",
			Owner:       "root:root",
		})
	}

	// update the pre- and post-commands

	preCommands := config.Spec.PreKubeadmCommands
	if preCommands == nil {
		preCommands = []string{}
	}

	postCommands := config.Spec.PostKubeadmCommands
	if postCommands == nil {
		postCommands = []string{}
	}

	preCommands = append(preCommands, "/etc/kube-vip-prepare.sh")
	postCommands = append(postCommands, "sed -i 's#path: /etc/kubernetes/super-admin.conf#path: /etc/kubernetes/admin.conf#' /etc/kubernetes/manifests/kube-vip.yaml")

	// add the post kubeadm command to replace the super-admin config
	return nil
}

func (r *KubeadmConfigWebhook) queryCluster(ctx context.Context, config *v1beta1.KubeadmConfig, clusterName string) (*infrav1.IonosCloudCluster, error) {
	logger := ctrl.LoggerFrom(ctx)

	var cluster clusterv1.Cluster
	clusterKey := client.ObjectKey{Namespace: config.Namespace, Name: clusterName}
	if err := r.Client.Get(ctx, clusterKey, &cluster); err != nil {
		logger.Error(err, "unable to retrieve management cluster", "cluster", clusterKey)
		return nil, err
	}

	var infraCluster infrav1.IonosCloudCluster
	clusterKey = client.ObjectKey{
		Namespace: cluster.Spec.InfrastructureRef.Namespace,
		Name:      cluster.Spec.InfrastructureRef.Name,
	}

	if err := r.Client.Get(ctx, clusterKey, &infraCluster); err != nil {
		logger.Error(err, "unable to retrieve infrastructure cluster", "cluster", clusterKey)
		return nil, err
	}

	return &infraCluster, nil
}

func (r *KubeadmConfigWebhook) queryLoadBalancer(ctx context.Context, infraCluster *infrav1.IonosCloudCluster) (*infrav1.IonosCloudLoadBalancer, error) {
	logger := ctrl.LoggerFrom(ctx)

	var loadBalancer infrav1.IonosCloudLoadBalancer
	loadBalancerKey := client.ObjectKey{
		Namespace: infraCluster.Namespace,
		Name:      infraCluster.Spec.LoadBalancerProviderRef.Name,
	}

	if err := r.Client.Get(ctx, loadBalancerKey, &loadBalancer); err != nil {
		logger.Error(err, "unable to retrieve load balancer", "load balancer", loadBalancerKey)
		return nil, err
	}

	return &loadBalancer, nil
}

// SetupWebhookWithManager will set up the manager to handle the webhooks.
func (r *KubeadmConfigWebhook) SetupWebhookWithManager(mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()

	return ctrl.NewWebhookManagedBy(mgr).
		For(&v1beta1.KubeadmConfig{}).
		WithDefaulter(r).
		Complete()
}
