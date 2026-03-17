# CLAUDE.md (k8s pod agent)

This is a Claude Code agent running inside a k3s pod.

## Build & Run

- Build: `cd rust && claude-k3 cargo-lock build --release 2>&1`
- Check: `cd rust && claude-k3 cargo-lock check 2>&1`
- Clippy: `cd rust && claude-k3 cargo-lock clippy --release -- -D warnings 2>&1`
- Fmt: `cd rust && claude-k3 cargo-lock fmt 2>&1`

## Rules

- Use `claude-k3 cargo-lock` for all cargo commands (build, check, clippy, fmt) to serialize builds across pods.
- CARGO_TARGET_DIR is set to /cargo-target (shared across all pods).
- Agent identity is derived from the workspace directory name.
- Git commits: always push immediately. Use concise, lowercase messages. Never include Co-Authored-By.
- Never use Unicode. Always use ASCII.
