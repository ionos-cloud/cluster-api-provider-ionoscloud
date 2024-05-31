#!/usr/bin/env bash

# Copyright 2024 IONOS Cloud.
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

set -o errexit
set -o pipefail
set -o nounset
if [[ "${TRACE-0}" == "1" ]]; then
    set -o xtrace
fi

# Directory where the files are located
DIR="_artifacts"

if [ -d "$DIR" ]; then
    echo "Directory \"$DIR\" exists. Proceeding with redacting sensitive information..."
else
    echo "Directory \"$DIR\" does not exist. Nothing to do!"
    exit 0
fi

# Value to replaced with
OLD_VALUE=$IONOS_TOKEN

# Value to replace the environment variable with
NEW_VALUE="[REDACTED]"

# Find all files in the directory and replace the environment variable with the new value
find "$DIR" -type f -exec perl -pi -e "s/\Q$OLD_VALUE\E/$NEW_VALUE/g" {} \;
