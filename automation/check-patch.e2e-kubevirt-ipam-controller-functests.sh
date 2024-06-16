#!/usr/bin/env bash

set -xeuE

# This script should be able to execute kubevirt ipam controller
# functional tests against Kubernetes cluster with
# CNAO built with latest changes, on any
# environment with basic dependencies listed in
# check-patch.packages installed and podman/docker running.
#
# dnf -y install automation/check-patch.packages
# automation/check-patch.e2e-kubevirt-ipam-controller-functests.sh

export KUBEVIRT_PROVIDER="kind-ovn"
export KUBEVIRTCI_TAG='2406041642-8d359a3'

# TODO KUBEVIRTCI_TAG

teardown() {
    cp $(find . -name "*junit*.xml") $ARTIFACTS || true
    rm -rf "${TMP_COMPONENT_PATH}"
    cd ${TMP_PROJECT_PATH}
    make cluster-down
}

main() {
    # Setup CNAO and artifacts temp directory
    source automation/check-patch.setup.sh
    cd ${TMP_PROJECT_PATH}

    # Spin-up ephemeral cluster with latest CNAO
    # this script also exports KUBECONFIG, and fetch $COMPONENT repository
    COMPONENT="kubevirt-ipam-controller" source automation/components-functests.setup.sh

    trap teardown EXIT

    ./hack/deploy-kubevirt.sh
    cd ${TMP_COMPONENT_PATH}
    echo "Run kubevirt-ipam-controller functional tests"
    make test-e2e
}

[[ "${BASH_SOURCE[0]}" == "$0" ]] && main "$@"
