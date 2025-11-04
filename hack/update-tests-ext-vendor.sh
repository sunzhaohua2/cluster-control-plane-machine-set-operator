#!/bin/bash

# This script updates the vendor directory for the tests-ext submodule.
# It handles both external dependencies and the parent module's local code.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TESTS_EXT_DIR="${REPO_ROOT}/test/control-plane-machine-set-tests-ext"

echo "==> Updating tests-ext vendor directory..."

# Change to tests-ext directory
cd "${TESTS_EXT_DIR}"

# Step 1: Run go mod vendor to get external dependencies
echo "  - Running 'go mod vendor' for external dependencies..."
go mod vendor

# Step 2: Sync parent module code (replace directive target)
echo "  - Syncing parent module code (test/e2e)..."
VENDOR_PARENT_DIR="vendor/github.com/openshift/cluster-control-plane-machine-set-operator"

# Remove old vendored parent module code
rm -rf "${VENDOR_PARENT_DIR}/test"

# Copy the parent module's test directory to vendor
mkdir -p "${VENDOR_PARENT_DIR}/test"
cp -r "${REPO_ROOT}/test/e2e" "${VENDOR_PARENT_DIR}/test/"

echo "✅ Vendor update complete!"
echo ""
echo "Summary:"
echo "  - External dependencies: vendored via 'go mod vendor'"
echo "  - Parent module (test/e2e): synced from ${REPO_ROOT}/test/e2e"
echo ""
echo "Vendor directory size:"
du -sh vendor
