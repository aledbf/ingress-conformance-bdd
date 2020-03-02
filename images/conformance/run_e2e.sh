#!/usr/bin/env bash

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

set -o errexit
set -o nounset
set -o pipefail

# Shutdown the tests gracefully then save the results
shutdown () {
    E2E_SUITE_PID=$(pgrep ingress-conformance-bdd.test)
    echo "sending TERM to ${E2E_SUITE_PID}"
    kill -s TERM "${E2E_SUITE_PID}"

    # Kind of a hack to wait for this pid to finish.
    # Since it's not a child of this shell we cannot use wait.
    tail --pid "${E2E_SUITE_PID}" -f /dev/null
    saveResults
}

saveResults() {
    cd "${RESULTS_DIR}" || exit
    tar -czf e2e.tar.gz ./*
    # mark the done file as a termination notice.
    echo -n "${RESULTS_DIR}/e2e.tar.gz" > "${RESULTS_DIR}/done"
}

# We get the TERM from kubernetes and handle it gracefully
trap shutdown TERM

set -x
/ust/local/bin/ingress-conformance-bdd.test -format cucumber "${RESULTS_DIR}"/ingress-conformance.json
ret=$?
set -x
saveResults
exit ${ret}
