#!/usr/bin/env bash

# Copyright 2023 The Kubernetes Authors.
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
set -o nounset
set -o pipefail
set -o xtrace

make test-e2e-install

cd "${GOPATH}"/src/k8s.io/kubernetes

kubetest2 kops -v=6 \
    --up --down --build --build-kubernetes=true --target-build-arch=linux/amd64 \
    --cloud-provider=gce --admin-access=0.0.0.0/0 \
    --kops-version-marker=https://storage.googleapis.com/kops-ci/bin/latest-ci.txt \
    --create-args "--networking=kubenet --set=spec.nodeProblemDetector.enabled=true" \
    --test=kops \
    -- \
    --ginkgo-args="--debug" \
    --use-built-binaries=true \
    --parallel=25
