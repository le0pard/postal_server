# ==========================================
# Stage 1: Builder
# ==========================================
FROM golang:1.26-bookworm AS builder

# args
ARG TARGETARCH
ARG BUILD_VERSION=
ARG BUILD_GIT_COMMIT=

# Install packages needed to build gems
RUN apt-get update -qq && apt-get install -yq --no-install-recommends \
  build-essential \
  curl \
  autoconf \
  automake \
  libtool \
  pkg-config \
  git \
  && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

# Clone, build, and install libpostal
RUN git clone https://github.com/openvenues/libpostal /code/libpostal
WORKDIR /code/libpostal
RUN ./bootstrap.sh && \
  ./configure --datadir=/usr/share/libpostal $([ "$TARGETARCH" = "arm64" ] && echo "--disable-sse2" || echo "") && \
  make -j4 && make check && make install && \
  ldconfig

# Create and change to the app directory.
WORKDIR /app

# Retrieve application dependencies.
# This allows the container build to reuse cached dependencies.
# Expecting to copy go.mod and if present go.sum.
COPY go.* ./
RUN go mod download

# Copy local code to the container image.
COPY . ./

# Build the binary.
ENV CGO_ENABLED=1
RUN go build -trimpath -ldflags="-s -w -X github.com/le0pard/postal_server/version.Version=$BUILD_VERSION -X github.com/le0pard/postal_server/version.GitCommit=$BUILD_GIT_COMMIT -X github.com/le0pard/postal_server/version.BuildTime=$(TZ=UTC date +"%Y-%m-%dT%H:%M:%S%z")" -v -o postal_server

# ==========================================
# Stage 2: Final Runtime
# ==========================================
# Use a slim debian image that matches the builder's OS (bookworm) to avoid glibc version mismatches
FROM debian:bookworm-slim

ARG GIN_MODE=release
ENV GIN_MODE=${GIN_MODE}

# Install minimal runtime dependencies (ca-certificates for external API calls if any, curl for healthchecks)
RUN apt-get update -qq && apt-get install -yq --no-install-recommends \
    ca-certificates \
    curl \
    && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

# Copy the compiled libpostal shared libraries from the builder
COPY --from=builder /usr/local/lib/libpostal.so* /usr/local/lib/

# Copy the downloaded libpostal machine learning data models
COPY --from=builder /usr/share/libpostal /usr/share/libpostal

# Update dynamic linker run-time bindings so the OS knows where libpostal.so is
RUN ldconfig

# Copy the compiled Go binary from the builder
WORKDIR /app
COPY --from=builder /app/postal_server /app/postal_server

EXPOSE 8000
# Run the web service on container startup.
CMD ["/app/postal_server"]
