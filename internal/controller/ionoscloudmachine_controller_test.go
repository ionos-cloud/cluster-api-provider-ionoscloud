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
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	conditions "sigs.k8s.io/cluster-api/util/conditions"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
)

func TestSetProvisioningCondition(t *testing.T) {
	machineWith := func(status metav1.ConditionStatus, reason string, provisioned bool) *infrav1.IonosCloudMachine {
		m := &infrav1.IonosCloudMachine{}
		if provisioned {
			m.Status.Initialization.Provisioned = new(true)
		}
		conditions.Set(m, metav1.Condition{Type: infrav1.MachineProvisionedCondition, Status: status, Reason: reason})
		return m
	}

	// A stale "waiting" reason is replaced once provisioning starts.
	m := machineWith(metav1.ConditionFalse, infrav1.WaitingForBootstrapDataReason, false)
	setProvisioningCondition(m, nil)
	c := conditions.Get(m, infrav1.MachineProvisionedCondition)
	require.Equal(t, metav1.ConditionFalse, c.Status)
	require.Equal(t, infrav1.MachineProvisioningReason, c.Reason)

	// A failing step is surfaced with its error.
	setProvisioningCondition(m, errors.New("error in step ReconcileServer: image lookup failed"))
	c = conditions.Get(m, infrav1.MachineProvisionedCondition)
	require.Equal(t, metav1.ConditionFalse, c.Status)
	require.Equal(t, infrav1.MachineProvisioningFailedReason, c.Reason)
	require.Equal(t, "error in step ReconcileServer: image lookup failed", c.Message)

	// A provisioned machine keeps its True condition on a transient error.
	m = machineWith(metav1.ConditionTrue, infrav1.MachineProvisionedReason, true)
	setProvisioningCondition(m, errors.New("transient"))
	c = conditions.Get(m, infrav1.MachineProvisionedCondition)
	require.Equal(t, metav1.ConditionTrue, c.Status)
	require.Equal(t, infrav1.MachineProvisionedReason, c.Reason)
}
