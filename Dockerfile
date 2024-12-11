# Use a compatible Go version
FROM golang:1.22-alpine

# Install dependencies
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
RUN mkdir -p /root/.git-monitor
# Set up working directory
WORKDIR /app

# Copy only the necessary files and directories
COPY server.go ./server.go
COPY internal ./internal
COPY config/git-monitor.yaml /home/appuser/.git-monitor.yaml
COPY data ./data
COPY bin/docker-vuln ./bin/docker-vuln
COPY .env .

RUN go mod init monitor
RUN go get go.mongodb.org/mongo-driver@latest
RUN go get github.com/joho/godotenv
RUN go mod tidy
# Make the binary in 'bin' executable
RUN chmod +x ./bin/docker-vuln

# Build the Go server
RUN go build -o server ./server.go

# Expose the port the server listens on
EXPOSE 8080

# Start the server
CMD ["./server"]
