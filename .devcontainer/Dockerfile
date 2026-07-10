# ── Dev Container ─────────────────────────────────────────────────────────────
# Full development environment: Go 1.23, Node.js 20, all CLI tools.
# Used by VS Code Dev Containers and GitHub Codespaces.
# NOT the production image — see ../Dockerfile for that.

FROM golang:1.26-bookworm

# ── System packages ───────────────────────────────────────────────────────────
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    wget \
    git \
    openssh-client \
    openssl \
    ca-certificates \
    gnupg \
    lsb-release \
    unzip \
    jq \
    make \
    bash-completion \
    && rm -rf /var/lib/apt/lists/*

# ── Node.js 20 (for React Email build) ───────────────────────────────────────
RUN curl -fsSL https://deb.nodesource.com/setup_20.x | bash - \
    && apt-get install -y --no-install-recommends nodejs \
    && rm -rf /var/lib/apt/lists/* \
    && node --version \
    && npm --version

# ── Docker CLI (so 'make docker-up' and 'make seed' work from inside devcontainer)
RUN curl -fsSL https://download.docker.com/linux/debian/gpg | gpg --dearmor -o /usr/share/keyrings/docker.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker.gpg] \
    https://download.docker.com/linux/debian $(lsb_release -cs) stable" \
    > /etc/apt/sources.list.d/docker.list \
    && apt-get update \
    && apt-get install -y --no-install-recommends docker-ce-cli docker-compose-plugin \
    && rm -rf /var/lib/apt/lists/*

# ── Go tools ──────────────────────────────────────────────────────────────────
ENV GOPATH=/go
ENV GOBIN=/go/bin
ENV PATH=/go/bin:/usr/local/go/bin:${PATH}

RUN go install golang.org/x/tools/cmd/goimports@latest \
    && go install github.com/air-verse/air@latest \
    && go install go.uber.org/mock/mockgen@latest \
    && echo "Go tools installed"

# ── golangci-lint ─────────────────────────────────────────────────────────────
RUN curl -sSfL https://golangci-lint.run/install.sh \
    | sh -s -- -b /go/bin v2.12.2 \
    && golangci-lint --version

# ── mongosh (MongoDB shell for manual inspection) ─────────────────────────────
RUN curl -fsSL https://www.mongodb.org/static/pgp/server-8.0.asc | gpg --dearmor \
    -o /usr/share/keyrings/mongodb-server-8.0.gpg \
    && echo "deb [ signed-by=/usr/share/keyrings/mongodb-server-8.0.gpg ] \
    https://repo.mongodb.org/apt/debian bookworm/mongodb-org/8.0 main" \
    > /etc/apt/sources.list.d/mongodb-org-8.0.list \
    && apt-get update \
    && apt-get install -y --no-install-recommends mongodb-mongosh \
    && rm -rf /var/lib/apt/lists/*

# ── redis-cli ─────────────────────────────────────────────────────────────────
RUN apt-get update && apt-get install -y --no-install-recommends redis-tools \
    && rm -rf /var/lib/apt/lists/*

# Install the Claude Code CLI globally
RUN npm install -g @anthropic-ai/claude-code

# Install the Gemini CLI globally
RUN npm install -g @google/gemini-cli

# Install the Codex CLI globally
RUN npm i -g @openai/codex

# ── Shell convenience ─────────────────────────────────────────────────────────
RUN echo 'alias ll="ls -la"' >> /root/.bashrc \
    && echo 'alias gs="git status"' >> /root/.bashrc \
    && echo 'export PS1="\[\033[01;32m\]devcontainer\[\033[00m\]:\[\033[01;34m\]\w\[\033[00m\]\$ "' \
    >> /root/.bashrc

WORKDIR /workspace
