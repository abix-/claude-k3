#!/usr/bin/env bash
# deploy.sh -- build the claude-agent image and deploy to k3s.
# Run inside WSL2: cd /mnt/c/code/k3s-claude && bash deploy.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
NERDCTL="sudo nerdctl --address /run/k3s/containerd/containerd.sock"
KUBECTL="sudo k3s kubectl"

echo "=== building claude-agent image ==="
${NERDCTL} build -t claude-agent:latest "${SCRIPT_DIR}"

echo "=== applying namespace ==="
${KUBECTL} apply -f "${SCRIPT_DIR}/namespace.yaml"

echo "=== applying PVCs ==="
${KUBECTL} apply -f "${SCRIPT_DIR}/pvc-cargo-target.yaml"
${KUBECTL} apply -f "${SCRIPT_DIR}/pvc-workspaces.yaml"

echo "=== creating configmap from scripts ==="
${KUBECTL} create configmap dispatcher-scripts -n claude-agents \
    --from-file=dispatcher.sh="${SCRIPT_DIR}/dispatcher.sh" \
    --from-file=job-template.yaml="${SCRIPT_DIR}/job-template.yaml" \
    --dry-run=client -o yaml | ${KUBECTL} apply -f -

echo "=== applying dispatcher cronjob + RBAC ==="
${KUBECTL} apply -f "${SCRIPT_DIR}/dispatcher-cronjob.yaml"

echo ""
echo "=== deployment complete ==="
echo ""
echo "next steps:"
echo "  1. create secret (if not already done):"
echo "     sudo k3s kubectl create secret generic claude-secrets -n claude-agents \\"
echo "       --from-literal=CLAUDE_CODE_OAUTH_TOKEN=<token> \\"
echo "       --from-literal=GITHUB_TOKEN=<token>"
echo ""
echo "  2. test dispatcher manually:"
echo "     sudo k3s kubectl create job --from=cronjob/claude-dispatcher test-dispatch -n claude-agents"
echo ""
echo "  3. watch pods:"
echo "     sudo k3s kubectl get pods -n claude-agents -w"
