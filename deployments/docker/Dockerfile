# --- Builder Stage ---
# This stage compiles the Go application into a static binary.
# Using a specific version of golang on alpine for a small and secure build environment.
FROM golang:1.21-alpine AS builder

# Install necessary build tools. git is needed to pull version information.
RUN apk --no-cache add git

# Set the working directory inside the container.
WORKDIR /app

# Copy go.mod and go.sum first to leverage Docker's layer caching.
# This step is only re-run if the dependencies change, speeding up subsequent builds.
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code into the container.
COPY . .

# Build the application.
# We use CGO_ENABLED=0 to create a static binary that can run on a minimal base image like alpine or scratch.
# The ldflags will be used to embed version information into the binary.
# In a real CI/CD pipeline, these ARG values would be passed in from the CI environment.
ARG VERSION="dev"
ARG GIT_COMMIT="unknown"
ARG BUILD_DATE="unknown"
ARG LDFLAGS="-s -w -X 'github.com/kubestack-ai/kubestack-ai/pkg/version.Version=${VERSION}' -X 'github.com/kubestack-ai/kubestack-ai/pkg/version.GitCommit=${GIT_COMMIT}' -X 'github.com/kubestack-ai/kubestack-ai/pkg/version.BuildDate=${BUILD_DATE}'"

RUN CGO_ENABLED=0 go build -ldflags="${LDFLAGS}" -o /ksa ./cmd/ksa


# --- Final Stage ---
# This stage creates the final, minimal production image.
# Using alpine as a base provides a good balance of small size and having a shell for debugging.
FROM alpine:latest

# It's a good practice to install certificates for making HTTPS requests to external APIs (like LLM providers).
RUN apk --no-cache add ca-certificates

# Create a non-root user and group for security. Running as root is a security risk.
RUN addgroup -S ksa && adduser -S ksa -G ksa

# Copy the compiled binary from the builder stage.
COPY --from=builder /ksa /usr/local/bin/ksa

# Copy the default configuration files into the image.
# Users can override these by mounting a volume at /etc/kubestack-ai.
COPY configs /etc/kubestack-ai

# Set correct ownership for the config directory.
RUN chown -R ksa:ksa /etc/kubestack-ai

# Switch to the non-root user.
USER ksa

# Define a health check for the container.
# This command simply checks if the binary is executable and can respond, which is a basic but effective health check.
HEALTHCHECK --interval=1m --timeout=5s --start-period=10s --retries=3 \
  CMD ksa version || exit 1

# Set the entrypoint for the container.
ENTRYPOINT ["ksa"]

# Set the default command to show the help message if no other command is provided.
CMD ["--help"]

# Personal.AI order the ending
