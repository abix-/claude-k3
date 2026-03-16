#!/usr/bin/env bash
set -euo pipefail

ISSUE_NUMBER="${ISSUE_NUMBER:?ISSUE_NUMBER env var required}"
AGENT_SLOT="${AGENT_SLOT:?AGENT_SLOT env var required}"
REPO_URL="${REPO_URL:-https://github.com/abix-/endless.git}"

AGENT_ID="claude-${AGENT_SLOT}"
WORKSPACE="/workspaces/endless-${AGENT_ID}"

export CARGO_TARGET_DIR="/cargo-target"

echo "[entrypoint] agent=${AGENT_ID} issue=${ISSUE_NUMBER}"

# set up workspace: clone once, fetch on reuse
if [ ! -d "${WORKSPACE}/.git" ]; then
    echo "[entrypoint] cloning repo into ${WORKSPACE}..."
    git clone "${REPO_URL}" "${WORKSPACE}"
else
    echo "[entrypoint] workspace exists, fetching..."
    git -C "${WORKSPACE}" fetch origin
fi

cd "${WORKSPACE}"

# configure git identity for this agent
git config user.name "${AGENT_ID}"
git config user.email "${AGENT_ID}@endless.dev"

# authenticate gh with the token from k8s secret
if [ -n "${GITHUB_TOKEN:-}" ]; then
    echo "${GITHUB_TOKEN}" | gh auth login --with-token
fi

echo "[entrypoint] launching claude for issue ${ISSUE_NUMBER}..."
exec claude --dangerously-skip-permissions -p "/issue ${ISSUE_NUMBER}"
