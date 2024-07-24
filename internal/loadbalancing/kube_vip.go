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
	"bytes"
	"errors"
	"net/netip"
	"text/template"
)

const defaultImage = "ghcr.io/kube-vip/kube-vip:v0.7.1"

const kubeVIPScript = `
#!/bin/bash

# Copyright 2020 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -e

# Configure the workaround required for kubeadm init with kube-vip:
# xref: https://github.com/kube-vip/kube-vip/issues/684

# Nothing to do for kubernetes < v1.29
KUBEADM_MINOR="$(kubeadm version -o short | cut -d '.' -f 2)"
if [[ "$KUBEADM_MINOR" -lt "29" ]]; then
	exit 0
fi

IS_KUBEADM_INIT="false"

# cloud-init kubeadm init
if [[ -f /run/kubeadm/kubeadm.yaml ]]; then
	IS_KUBEADM_INIT="true"
fi

# ignition kubeadm init
if [[ -f /etc/kubeadm.sh ]] && grep -q -e "kubeadm init" /etc/kubeadm.sh; then
	IS_KUBEADM_INIT="true"
fi

if [[ "$IS_KUBEADM_INIT" == "true" ]]; then
	sed -i 's#path: /etc/kubernetes/admin.conf#path: /etc/kubernetes/super-admin.conf#' \
	  /etc/kubernetes/manifests/kube-vip.yaml
fi
`

const kubeVIPTemplate = `
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
          value: '{{ .VIPInterface }}'
        - name: address
          value: '{{ .VIPAddress }}'
        - name: port
          value: "{{ .VIPPort }}"
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
      image: {{ .VIPImage }}
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

// kubeVIPConfig defines the configuration for kube-vip.
type kubeVIPConfig struct {
	VIPAddress   string
	VIPPort      int
	VIPImage     string
	VIPInterface string
}

// RenderKubeVIPConfig renders the kube-vip configuration based on the provided parameters.
func RenderKubeVIPConfig(host string, port int, image, iface string) (string, error) {
	config, err := newKubeVIPConfig(host, port, image, iface)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := vipTemplate.Execute(&buf, config); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// newKubeVIPConfig creates a new kubeVIPConfig.
func newKubeVIPConfig(hostIP string, port int, image, networkInterface string) (*kubeVIPConfig, error) {
	if _, err := netip.ParseAddr(hostIP); err != nil {
		return nil, errors.New("VIP IP address is required and needs to be a valid IP address")
	}

	if port < 1024 || port > 65535 {
		return nil, errors.New("VIP port is required and needs to be a valid port")
	}

	if image == "" {
		image = defaultImage
	}

	return &kubeVIPConfig{
		VIPAddress:   hostIP,
		VIPPort:      port,
		VIPImage:     image,
		VIPInterface: networkInterface,
	}, nil
}

// KubeVIPScript returns the script to configure kube-vip on higher Kubernetes versions.
func KubeVIPScript() string {
	return kubeVIPScript
}

var vipTemplate *template.Template

func init() {
	vipTemplate = template.Must(template.New("kube-vip").Parse(kubeVIPTemplate))
}
