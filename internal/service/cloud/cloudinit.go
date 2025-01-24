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
	"encoding/base64"
	"fmt"
)

func enrichCloudInitConfig(machineName string, userData string) string {
	const bootCmdFormat = `bootcmd:
  - echo %[1]s > /etc/hostname
  - hostname %[1]s
`
	bootCmdString := fmt.Sprintf(bootCmdFormat, machineName)
	enrichedData := fmt.Sprintf("%s\n%s", userData, bootCmdString)

	return base64.StdEncoding.EncodeToString([]byte(enrichedData))
}
