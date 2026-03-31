# Build stage
ARG go_version=1.23.8
ARG alpine_version=3.23
FROM --platform=$BUILDPLATFORM golang:${go_version}-alpine${alpine_version} AS builder

# Set the working directory
WORKDIR /build

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
ARG TARGETARCH
ARG TARGETOS
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o server ./cmd/web

# Final stage
FROM alpine:${alpine_version}

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /build/server /app/server

# Create uploads directory
RUN mkdir -p uploads

# Expose port 8080
EXPOSE 8080

# Set the uploads directory as a volume
VOLUME ["/app/uploads"]

# Run the application
CMD ["/app/server"] 
