/*
Copyright 2025 IONOS Cloud.

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

package cloud

import (
	"encoding/json"
	"fmt"

	ignition "github.com/flatcar/ignition/config/v2_3"
	igntypes "github.com/flatcar/ignition/config/v2_3/types"
	"k8s.io/utils/ptr"

	infrav1 "github.com/ionos-cloud/cluster-api-provider-ionoscloud/api/v1alpha1"
)

func enrichIgnitionConfig(machine *infrav1.IonosCloudMachine, bootstrapData []byte) ([]byte, error) {
	ign, err := convertToIgnition(bootstrapData)
	if err != nil {
		return nil, fmt.Errorf("converting bootstrap-data to Ignition: %w", err)
	}

	baseConfig := getIgnitionBaseConfig(machine)
	ign = ignition.Append(ign, baseConfig)

	userData, err := json.Marshal(&ign)
	if err != nil {
		return nil, fmt.Errorf("marshaling generated Ignition config into JSON: %w", err)
	}

	return userData, nil
}

func convertToIgnition(data []byte) (igntypes.Config, error) {
	cfg, reports, err := ignition.Parse(data)
	if err != nil {
		return igntypes.Config{}, fmt.Errorf("parsing Ignition config: %w", err)
	}
	if reports.IsFatal() {
		return igntypes.Config{}, fmt.Errorf("error parsing Ignition config: %v", reports.String())
	}

	return cfg, nil
}

func getIgnitionBaseConfig(machine *infrav1.IonosCloudMachine) igntypes.Config {
	metadata := fmt.Sprintf("data:,COREOS_CUSTOM_HOSTNAME=%s\n", machine.Name)
	return igntypes.Config{
		Storage: igntypes.Storage{
			Files: []igntypes.File{
				{
					Node: igntypes.Node{
						Filesystem: "root",
						Path:       "/etc/hostname",
						Overwrite:  ptr.To(true),
					},
					FileEmbedded1: igntypes.FileEmbedded1{
						Mode: ptr.To(0o644),
						Contents: igntypes.FileContents{
							Source: "data:," + machine.Name,
						},
					},
				},
				{
					Node: igntypes.Node{
						Filesystem: "root",
						Path:       "/etc/ionoscloud-env",
						Overwrite:  ptr.To(true),
					},
					FileEmbedded1: igntypes.FileEmbedded1{
						Mode: ptr.To(420),
						Contents: igntypes.FileContents{
							Source: metadata,
						},
					},
				},
			},
		},
		Systemd: igntypes.Systemd{
			Units: []igntypes.Unit{
				{
					Name:   "systemd-resolved.service",
					Enable: true,
				},
			},
		},
	}
}
