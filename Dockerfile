# Use a compatible Go version for osv-scanner
FROM golang:1.22-alpine

# Install dependencies, including sudo
RUN apk add --no-cache \
    bash \
    docker \
    openrc \
    curl \
    git \
    sudo

# Add a non-root user (optional, for better security)
RUN adduser -D -s /bin/bash appuser && \
    echo "appuser ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers

# Install Syft using its installation script (run as root)
USER root
RUN curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh | sh -s -- -b /usr/local/bin

# Install git-monitor
RUN go install -ldflags "-X main.BuildDate=$(date +'%Y-%m-%dT%H:%M:%S%z')" github.com/kriechi/git-monitor@latest

# Install osv-scanner
RUN go install github.com/google/osv-scanner/cmd/osv-scanner@latest

# Set up working directory and Docker monitor path
RUN mkdir -p /root/.git-monitor

WORKDIR /app

# Copy the Go server code, repos.txt, and CLI binary
COPY server.go . 
COPY repos.txt . 
COPY docker-vuln ./docker-vuln
COPY git-monitor.yaml /home/appuser/.git-monitor.yaml

# Make the CLI tool executable
RUN chmod +x ./docker-vuln

# Build the Go server
RUN go build -o server server.go

# Expose the port the server listens on
EXPOSE 8080

# Start the server and Docker daemon
CMD ["sh", "-c", "sudo dockerd & ./server"]
