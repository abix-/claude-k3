#!/usr/bin/env bash
# dispatcher.sh -- find eligible GitHub issues and create k8s Jobs for them.
# Runs as a CronJob every 3 minutes inside the claude-agents namespace.
set -euo pipefail

REPO="abix-/endless"
NAMESPACE="claude-agents"
MAX_SLOTS=10
JOB_TEMPLATE="/etc/dispatcher/job-template.yaml"

echo "[dispatcher] $(date -Iseconds) starting scan"

# get issues that are ready or needs-review (eligible for work)
READY_ISSUES=$(gh issue list --repo "${REPO}" --label ready --state open --json number --jq '.[].number' 2>/dev/null || echo "")
REVIEW_ISSUES=$(gh issue list --repo "${REPO}" --label needs-review --state open --json number --jq '.[].number' 2>/dev/null || echo "")

# needs-review takes priority (matches /issue claim algorithm)
ALL_ISSUES=""
if [ -n "${REVIEW_ISSUES}" ]; then
    ALL_ISSUES="${REVIEW_ISSUES}"
fi
if [ -n "${READY_ISSUES}" ]; then
    if [ -n "${ALL_ISSUES}" ]; then
        ALL_ISSUES="${ALL_ISSUES}"$'\n'"${READY_ISSUES}"
    else
        ALL_ISSUES="${READY_ISSUES}"
    fi
fi

if [ -z "${ALL_ISSUES}" ]; then
    echo "[dispatcher] no eligible issues found"
    exit 0
fi

echo "[dispatcher] eligible issues: $(echo "${ALL_ISSUES}" | tr '\n' ' ')"

# get currently active jobs (running or pending)
ACTIVE_ISSUES=$(kubectl get jobs -n "${NAMESPACE}" -l app=claude-agent \
    --field-selector=status.active=1 \
    -o jsonpath='{.items[*].metadata.labels.issue-number}' 2>/dev/null || echo "")
ACTIVE_SLOTS=$(kubectl get jobs -n "${NAMESPACE}" -l app=claude-agent \
    --field-selector=status.active=1 \
    -o jsonpath='{.items[*].metadata.labels.agent-slot}' 2>/dev/null || echo "")

ACTIVE_COUNT=$(echo "${ACTIVE_SLOTS}" | wc -w | tr -d ' ')
echo "[dispatcher] active jobs: ${ACTIVE_COUNT}, slots in use: ${ACTIVE_SLOTS}"

# clean up completed/failed jobs older than 1 hour
kubectl get jobs -n "${NAMESPACE}" -l app=claude-agent \
    -o jsonpath='{range .items[?(@.status.active!=1)]}{.metadata.name}{"\n"}{end}' 2>/dev/null | \
    while read -r job_name; do
        [ -z "${job_name}" ] && continue
        kubectl delete job "${job_name}" -n "${NAMESPACE}" --ignore-not-found 2>/dev/null || true
    done

# create jobs for eligible issues
for ISSUE in ${ALL_ISSUES}; do
    # skip if already being worked
    if echo "${ACTIVE_ISSUES}" | grep -qw "${ISSUE}"; then
        echo "[dispatcher] issue ${ISSUE} already has an active job, skipping"
        continue
    fi

    # check max concurrency
    if [ "${ACTIVE_COUNT}" -ge "${MAX_SLOTS}" ]; then
        echo "[dispatcher] at max capacity (${MAX_SLOTS}), stopping"
        break
    fi

    # find a free slot
    SLOT=""
    for i in $(seq 1 ${MAX_SLOTS}); do
        if ! echo "${ACTIVE_SLOTS}" | grep -qw "${i}"; then
            SLOT="${i}"
            break
        fi
    done

    if [ -z "${SLOT}" ]; then
        echo "[dispatcher] no free slots available"
        break
    fi

    echo "[dispatcher] creating job for issue ${ISSUE} in slot ${SLOT}"

    # generate job manifest from template
    sed -e "s/ISSUE_NUMBER/${ISSUE}/g" -e "s/AGENT_SLOT/${SLOT}/g" \
        "${JOB_TEMPLATE}" | kubectl apply -n "${NAMESPACE}" -f -

    ACTIVE_COUNT=$((ACTIVE_COUNT + 1))
    ACTIVE_SLOTS="${ACTIVE_SLOTS} ${SLOT}"
    ACTIVE_ISSUES="${ACTIVE_ISSUES} ${ISSUE}"
done

echo "[dispatcher] $(date -Iseconds) scan complete"
