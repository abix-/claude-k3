FROM ubuntu:24.04

ENV DEBIAN_FRONTEND=noninteractive

# system deps
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl ca-certificates git python3 build-essential pkg-config \
    libssl-dev unzip jq && \
    rm -rf /var/lib/apt/lists/*

# node.js 22 LTS
RUN curl -fsSL https://deb.nodesource.com/setup_22.x | bash - && \
    apt-get install -y nodejs && \
    rm -rf /var/lib/apt/lists/*

# claude code CLI
RUN npm install -g @anthropic-ai/claude-code

# github CLI
RUN curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg \
    -o /usr/share/keyrings/githubcli-archive-keyring.gpg && \
    echo "deb [arch=amd64 signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" \
    > /etc/apt/sources.list.d/github-cli.list && \
    apt-get update && apt-get install -y gh && \
    rm -rf /var/lib/apt/lists/*

# rust toolchain (as non-root user later)
ENV RUSTUP_HOME=/usr/local/rustup
ENV CARGO_HOME=/usr/local/cargo
ENV PATH="/usr/local/cargo/bin:${PATH}"
RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | \
    sh -s -- -y --default-toolchain stable --profile minimal && \
    rustup component add clippy rustfmt

# cargo-lock serializer (linux version)
COPY cargo-lock-linux.py /usr/local/bin/cargo-lock.py
RUN chmod +x /usr/local/bin/cargo-lock.py

# entrypoint
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# claude config -- baked into image for v1
# skills and CLAUDE.md are copied at build time
COPY claude-config/ /root/.claude/

WORKDIR /workspaces

ENTRYPOINT ["/entrypoint.sh"]
