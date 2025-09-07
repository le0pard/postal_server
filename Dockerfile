# Use the offical golang image to create a binary.
# This is based on Debian and sets the GOPATH to /go.
# https://hub.docker.com/_/golang
FROM golang:1.25

# args
ARG TARGETARCH
ARG BUILD_VERSION=
ARG BUILD_GIT_COMMIT=
ARG GIN_MODE=release

# Set up environment variables
ENV GIN_MODE=${GIN_MODE}

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

# install libpostal

RUN echo $TARGETARCH
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
RUN go build -trimpath -ldflags="-s -w -X github.com/le0pard/postal_server/version.Version=$BUILD_VERSION -X github.com/le0pard/postal_server/version.GitCommit=$BUILD_GIT_COMMIT -X github.com/le0pard/postal_server/version.BuildTime=$(TZ=UTC date +"%Y-%m-%dT%H:%M:%S%z")" -v -o postal_server

EXPOSE 8000
# Run the web service on container startup.
CMD ["/app/postal_server"]
