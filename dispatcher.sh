#!/usr/bin/env bash
# dispatcher.sh -- find eligible GitHub issues and create k8s Jobs for them.
# Runs as a CronJob every 3 minutes inside the claude-agents namespace.
#
# GitHub labels are the source of truth. The dispatcher only reads labels
# to find eligible issues. It does NOT use job existence for dedup --
# once a pod claims an issue, the label changes to 'claimed' and the
# issue drops out of the eligible list naturally.
set -euo pipefail

REPO="abix-/endless"
NAMESPACE="claude-agents"
MAX_SLOTS=3
JOB_TEMPLATE="/etc/dispatcher/job-template.yaml"
TIMESTAMP=$(date +%s)

echo "[dispatcher] $(TZ=America/New_York date '+%Y-%m-%d %H:%M:%S %Z') starting scan"

# get issues that are ready or needs-review (eligible for work)
READY_ISSUES=$(gh issue list --repo "${REPO}" --label ready --state open --json number --jq '[.[].number] | sort | .[]' 2>/dev/null || echo "")
REVIEW_ISSUES=$(gh issue list --repo "${REPO}" --label needs-review --state open --json number --jq '[.[].number] | sort | .[]' 2>/dev/null || echo "")

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

# get currently active (running) jobs -- only these consume slots
ACTIVE_SLOTS=$(kubectl get jobs -n "${NAMESPACE}" -l app=claude-agent \
    --field-selector=status.active=1 \
    -o jsonpath='{.items[*].metadata.labels.agent-slot}' 2>/dev/null || echo "")

ACTIVE_COUNT=$(echo "${ACTIVE_SLOTS}" | wc -w | tr -d ' ')
echo "[dispatcher] active jobs: ${ACTIVE_COUNT}, slots in use: ${ACTIVE_SLOTS}"

# create jobs for eligible issues
for ISSUE in ${ALL_ISSUES}; do
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

    # unique job name so completed jobs don't collide
    sed -e "s/__ISSUE_NUMBER__/${ISSUE}/g" -e "s/__AGENT_SLOT__/${SLOT}/g" \
        "${JOB_TEMPLATE}" | \
        sed "s/name: \"claude-issue-${ISSUE}\"/name: \"claude-issue-${ISSUE}-${TIMESTAMP}\"/" | \
        kubectl apply -n "${NAMESPACE}" -f -

    ACTIVE_COUNT=$((ACTIVE_COUNT + 1))
    ACTIVE_SLOTS="${ACTIVE_SLOTS} ${SLOT}"
done

echo "[dispatcher] $(TZ=America/New_York date '+%Y-%m-%d %H:%M:%S %Z') scan complete"
