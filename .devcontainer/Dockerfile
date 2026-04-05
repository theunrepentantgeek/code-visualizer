# See here for image contents: https://github.com/devcontainers/images/blob/main/src/go/.devcontainer/Dockerfile

# Compound versioning - v2 of the devcontainer, v1.26 of Go. 
# Should be automatically kept up to date by dependabot.
FROM mcr.microsoft.com/devcontainers/go:2-1.26

# APT dependencies
RUN apt-get update \
    && apt-get -y install --no-install-recommends bash-completion graphviz zip python3-pip

# Install development tools
COPY .devcontainer/install-dependencies.sh .
RUN ./install-dependencies.sh devcontainer && rm install-dependencies.sh

# Setup go-task completions
RUN curl -sL "https://raw.githubusercontent.com/go-task/task/v3.2.0/completion/bash/task.bash" > "/etc/.task.completion.sh" \
    && echo 'source /etc/.task.completion.sh' >> "/etc/bash.bashrc"

# Pre-download Go modules for faster builds
COPY go.mod go.sum /tmp/mod-download/
RUN cd /tmp/mod-download && go mod download && rm -rf /tmp/mod-download
