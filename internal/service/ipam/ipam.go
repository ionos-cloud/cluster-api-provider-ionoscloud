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

// Package ipam offers services for IPAM management.
package ipam

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ipamv1 "sigs.k8s.io/cluster-api/exp/ipam/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
	"github.com/ionos-cloud/cluster-api-provider-ionoscloud/scope"
)

const (
	// PrimaryNICFormat is the format used for IPAddressClaims for the primary nic.
	PrimaryNICFormat = "nic-%s"

	// AdditionalNICFormat is the format used for IPAddressClaims for additional nics.
	AdditionalNICFormat = "nic-%s-%d"

	// IPV4Format is the IP v4 format.
	IPV4Format = "ipv4"

	// IPV6Format is the IP v6 format.
	IPV6Format = "ipv6"
)

// Helper offers IP address management services for IONOS Cloud machine reconciliation.
type Helper struct {
	logger logr.Logger
	client client.Client
}

// NewHelper creates new Helper.
func NewHelper(c client.Client, log logr.Logger) *Helper {
	h := new(Helper)
	h.client = c
	h.logger = log

	return h
}

// ReconcileIPAddresses prevents successful reconciliation of a IonosCloudMachine
// until an IPAMConfig Provider updates each IPAddressClaim associated to the
// IonosCloudMachine with a reference to an IPAddress. The IPAddress is stored in the status.
// This function is a no-op if the IonosCloudMachine has no associated IPAddressClaims.
func (h *Helper) ReconcileIPAddresses(ctx context.Context, machineScope *scope.Machine) (requeue bool, err error) {
	log := h.logger.WithName("reconcileIPAddresses")
	log.V(4).Info("reconciling IPAddresses.")

	networkInfos := &[]infrav1.NICInfo{}

	// primary NIC.
	requeue, err = h.handlePrimaryNIC(ctx, machineScope, networkInfos)
	if err != nil {
		return true, errors.Join(err, errors.New("unable to handle primary nic"))
	}

	if machineScope.IonosMachine.Spec.AdditionalNetworks != nil {
		waitForAdditionalIP, err := h.handleAdditionalNICs(ctx, machineScope, networkInfos)
		if err != nil {
			return true, errors.Join(err, errors.New("unable to handle additional nics"))
		}
		requeue = requeue || waitForAdditionalIP
	}

	// update the status
	log.V(4).Info("updating IonosMachine.status.machineNetworkInfo.")
	machineScope.IonosMachine.Status.MachineNetworkInfo = &infrav1.MachineNetworkInfo{NICInfo: *networkInfos}

	return requeue, nil
}

func (h *Helper) ReconcileIPAddressClaimsDeletion(ctx context.Context, machineScope *scope.Machine) (err error) {
	log := h.logger.WithName("reconcileIPAddressClaimsDeletion")
	log.V(4).Info("removing finalizers from IPAddressClaims.")

	formats := []string{IPV4Format, IPV6Format}
	nicNames := []string{fmt.Sprintf(PrimaryNICFormat, machineScope.IonosMachine.Name)}

	for _, network := range machineScope.IonosMachine.Spec.AdditionalNetworks {
		nicName := fmt.Sprintf(AdditionalNICFormat, machineScope.IonosMachine.Name, network.NetworkID)
		nicNames = append(nicNames, nicName)
	}

	for _, format := range formats {
		for _, nicName := range nicNames {
			key := client.ObjectKey{
				Namespace: machineScope.IonosMachine.Namespace,
				Name:      fmt.Sprintf("%s-%s", nicName, format),
			}

			claim, err := h.GetIPAddressClaim(ctx, key)
			if err != nil {
				if apierrors.IsNotFound(err) {
					continue
				}
				return err
			}

			if updated := controllerutil.RemoveFinalizer(claim, infrav1.MachineFinalizer); updated {
				if err = h.client.Update(ctx, claim); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (h *Helper) handlePrimaryNIC(ctx context.Context, machineScope *scope.Machine, nics *[]infrav1.NICInfo) (waitForIP bool, err error) {
	nic := infrav1.NICInfo{Primary: true}
	ipamConfig := machineScope.IonosMachine.Spec.IPAMConfig
	nicName := fmt.Sprintf(PrimaryNICFormat, machineScope.IonosMachine.Name)

	// default NIC ipv4.
	if ipamConfig.IPv4PoolRef != nil {
		ip, err := h.handleIPAddressForNIC(ctx, machineScope, nicName, IPV4Format, ipamConfig.IPv4PoolRef)
		if err != nil {
			return false, err
		}
		if ip == "" {
			waitForIP = true
		} else {
			nic.IPv4Addresses = []string{ip}
		}
	}

	// default NIC ipv6.
	if ipamConfig.IPv6PoolRef != nil {
		ip, err := h.handleIPAddressForNIC(ctx, machineScope, nicName, IPV6Format, ipamConfig.IPv6PoolRef)
		if err != nil {
			return false, err
		}
		if ip == "" {
			waitForIP = true
		} else {
			nic.IPv6Addresses = []string{ip}
		}
	}

	*nics = append(*nics, nic)

	return waitForIP, nil
}

func (h *Helper) handleAdditionalNICs(ctx context.Context, machineScope *scope.Machine, nics *[]infrav1.NICInfo) (waitForIP bool, err error) {
	for _, net := range machineScope.IonosMachine.Spec.AdditionalNetworks {
		nic := infrav1.NICInfo{Primary: false}
		nicName := fmt.Sprintf(AdditionalNICFormat, machineScope.IonosMachine.Name, net.NetworkID)
		if net.IPv4PoolRef != nil {
			ip, err := h.handleIPAddressForNIC(ctx, machineScope, nicName, IPV4Format, net.IPv4PoolRef)
			if err != nil {
				return false, errors.Join(err, fmt.Errorf("unable to handle IPv4Address for nic %s", nicName))
			}
			if ip == "" {
				waitForIP = true
			} else {
				nic.IPv4Addresses = []string{ip}
			}
		}

		if net.IPv6PoolRef != nil {
			ip, err := h.handleIPAddressForNIC(ctx, machineScope, nicName, IPV6Format, net.IPv6PoolRef)
			if err != nil {
				return false, errors.Join(err, fmt.Errorf("unable to handle IPv6Address for nic %s", nicName))
			}
			if ip == "" {
				waitForIP = true
			} else {
				nic.IPv6Addresses = []string{ip}
			}
		}

		*nics = append(*nics, nic)
	}

	return waitForIP, nil
}

// handleIPAddressForNIC checks for an IPAddressClaim. If there is one it extracts the ip from the corresponding IPAddress object, otherwise it creates the IPAddressClaim and returns early.
func (h *Helper) handleIPAddressForNIC(ctx context.Context, machineScope *scope.Machine, nic, suffix string, poolRef *corev1.TypedLocalObjectReference) (ip string, err error) {
	log := h.logger.WithName("handleIPAddressForNIC")

	key := client.ObjectKey{
		Namespace: machineScope.IonosMachine.Namespace,
		Name:      fmt.Sprintf("%s-%s", nic, suffix),
	}

	claim, err := h.GetIPAddressClaim(ctx, key)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return "", err
		}
		log.V(4).Info("IPAddressClaim not found, creating it.", "nic", nic)
		err = h.CreateIPAddressClaim(ctx, machineScope.IonosMachine, key.Name, poolRef)
		if err != nil {
			return "", errors.Join(err, fmt.Errorf("unable to create IPAddressClaim for machine %s", machineScope.IonosMachine.Name))
		}
		// we just created the claim, so we can return early and wait for the creation of the IPAddress.
		return "", nil
	}

	// we found a claim, lets see if there is an IPAddress
	ipAddrName := claim.Status.AddressRef.Name
	if ipAddrName == "" {
		log.V(4).Info("No IPAddress found yet.", "nic", nic)
		return "", nil
	}

	ipAddrKey := types.NamespacedName{
		Namespace: machineScope.IonosMachine.Namespace,
		Name:      ipAddrName,
	}
	ipAddr, err := h.GetIPAddress(ctx, ipAddrKey)
	if err != nil {
		return "", errors.Join(err, fmt.Errorf("unable to get IPAddress specified in claim %s", claim.Name))
	}

	ip = ipAddr.Spec.Address

	log.V(4).Info("IPAddress found, ", "ip", ip, "nic", nic)

	return ip, nil
}

// CreateIPAddressClaim creates an IPAddressClaim for a given object.
func (h *Helper) CreateIPAddressClaim(ctx context.Context, owner client.Object, name string, poolRef *corev1.TypedLocalObjectReference) error {
	claimRef := types.NamespacedName{
		Namespace: owner.GetNamespace(),
		Name:      name,
	}

	ipAddrClaim := &ipamv1.IPAddressClaim{}
	var err error
	if err = h.client.Get(ctx, claimRef, ipAddrClaim); err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	if !apierrors.IsNotFound(err) {
		// IPAddressClaim already exists
		return nil
	}

	desired := &ipamv1.IPAddressClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      claimRef.Name,
			Namespace: claimRef.Namespace,
		},
		Spec: ipamv1.IPAddressClaimSpec{
			PoolRef: *poolRef,
		},
	}
	_, err = controllerutil.CreateOrUpdate(ctx, h.client, desired, func() error {
		controllerutil.AddFinalizer(desired, infrav1.MachineFinalizer)
		return controllerutil.SetControllerReference(owner, desired, h.client.Scheme())
	})

	return err
}

// GetIPAddress attempts to retrieve the IPAddress.
func (h *Helper) GetIPAddress(ctx context.Context, key client.ObjectKey) (*ipamv1.IPAddress, error) {
	out := &ipamv1.IPAddress{}
	err := h.client.Get(ctx, key, out)
	if err != nil {
		return nil, err
	}

	return out, nil
}

// GetIPAddressClaim attempts to retrieve the IPAddressClaim.
func (h *Helper) GetIPAddressClaim(ctx context.Context, key client.ObjectKey) (*ipamv1.IPAddressClaim, error) {
	out := &ipamv1.IPAddressClaim{}
	err := h.client.Get(ctx, key, out)
	if err != nil {
		return nil, err
	}

	return out, nil
}
