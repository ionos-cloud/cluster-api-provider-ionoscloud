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

package loadbalancing

import (
	"errors"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

const defaultTemplate = `
---
apiVersion: v1
kind: Pod
metadata:
  name: kube-vip
  namespace: kube-system
spec:
  containers:
    - args:
        - manager
      env:
        - name: cp_enable
          value: 'true'
        - name: vip_interface
          value: '%s'
        - name: address
          value: '%s'
        - name: port
          value: "%d"
        - name: vip_arp
          value: 'true'
        - name: vip_leaderelection
          value: 'true'
        - name: vip_leaseduration
          value: '15'
        - name: vip_renewdeadline
          value: '10'
        - name: vip_retryperiod
          value: '2'
      image: %s
      imagePullPolicy: IfNotPresent
      name: kube-vip
      resources: {}
      securityContext:
        capabilities:
          add:
            - NET_ADMIN
            - NET_RAW
      volumeMounts:
        - mountPath: /etc/kubernetes/admin.conf
          name: kubeconfig
  hostAliases:
    - hostnames:
        - kubernetes
        - localhost
      ip: 127.0.0.1
  hostNetwork: true
  volumes:
    - hostPath:
        path: /etc/kubernetes/admin.conf
        type: FileOrCreate
      name: kubeconfig
status: {}
}
`

const defaultIP = "192.0.2.0"

func formatExpected(networkInterface, ip string, port int, image string) string {
	if image == "" {
		image = defaultImage
	}

	return fmt.Sprintf(defaultTemplate, networkInterface, ip, port, image)
}

func TestRenderKubeVIPConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      kubeVIPConfig
		expected    string
		expectedErr error
	}{{
		name: "Render with default image",
		config: kubeVIPConfig{
			VIPAddress:   defaultIP,
			VIPPort:      6443,
			VIPImage:     "",
			VIPInterface: "",
		},
		expected: formatExpected("", defaultIP, 6443, ""),
	}, {
		name: "Render with custom image",
		config: kubeVIPConfig{
			VIPAddress:   defaultIP,
			VIPPort:      6443,
			VIPImage:     "custom-image",
			VIPInterface: "",
		},
		expected: formatExpected("", defaultIP, 6443, "custom-image"),
	}, {
		name: "Render with custom interface",
		config: kubeVIPConfig{
			VIPAddress:   defaultIP,
			VIPPort:      6443,
			VIPImage:     "",
			VIPInterface: "eth0",
		},
		expected: formatExpected("eth0", defaultIP, 6443, ""),
	}, {
		name: "error for invalid port",
		config: kubeVIPConfig{
			VIPAddress:   defaultIP,
			VIPPort:      0,
			VIPImage:     "",
			VIPInterface: "",
		},
		expectedErr: errors.New("VIP port is required and needs to be a valid port"),
	}, {
		name: "error for invalid IP",
		config: kubeVIPConfig{
			VIPAddress:   "invalid",
			VIPPort:      6443,
			VIPImage:     "",
			VIPInterface: "",
		},
		expectedErr: errors.New("VIP IP address is required and needs to be a valid IP address"),
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual, err := RenderKubeVIPConfig(tt.config.VIPAddress, tt.config.VIPPort, tt.config.VIPImage, tt.config.VIPInterface)

			if tt.expectedErr != nil {
				require.EqualError(t, err, tt.expectedErr.Error())
			} else {
				require.NoError(t, err)
				diff := cmp.Diff(tt.expected, actual)
				require.Empty(t, diff)
			}
		})
	}
}
