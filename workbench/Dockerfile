# Use Ubuntu 24.04 as the base image
FROM ubuntu:24.04

# Avoid prompts from apt
ENV DEBIAN_FRONTEND=noninteractive

# Update and install dependencies
RUN apt-get update && apt-get install -y \
    git \
    make \
    wget \
    apt-transport-https \
    lsb-release \
    ca-certificates \
    curl \
    && rm -rf /var/lib/apt/lists/*


# Install Docker Engine 26.1.4 without starting the service
RUN install -m 0755 -d /etc/apt/keyrings && \
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc && \
    chmod a+r /etc/apt/keyrings/docker.asc && \
    echo \
    "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu \
    $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | \
    tee /etc/apt/sources.list.d/docker.list > /dev/null && \
    apt-get update && \
    VERSION_STRING=5:26.1.4-1~ubuntu.24.04~noble && \
    apt-get install -y --no-install-recommends \
    docker-ce=$VERSION_STRING \
    docker-ce-cli=$VERSION_STRING \
    containerd.io \
    docker-buildx-plugin \
    docker-compose-plugin && \
    rm -rf /var/lib/apt/lists/*

# Install Go 1.23.1
ENV GO_VERSION=1.23.1
RUN wget https://golang.org/dl/go${GO_VERSION}.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz && \
    rm go${GO_VERSION}.linux-amd64.tar.gz
# Set up Go environment
ENV GOPATH=/go
ENV PATH=$GOPATH/bin:/usr/local/go/bin:$PATH
ENV GOFLAGS=-buildvcs=false
# Install golangci-lint from source
RUN go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.61.0
RUN set -e
# Set the working directory in the container
WORKDIR /app

# Create a wrapper script to start Docker and then run the main command
COPY start.sh /start.sh
RUN chmod +x /start.sh

# Use the wrapper script as the entry point
ENTRYPOINT ["/start.sh"]
CMD ["/bin/bash"]