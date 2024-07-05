# Use the offical golang image to create a binary.
# This is based on Debian and sets the GOPATH to /go.
# https://hub.docker.com/_/golang
FROM golang:1.22-bookworm as builder

# Install packages needed to build gems
RUN apt-get update -qq && apt-get install -yq --no-install-recommends \
  build-essential \
  curl \
  autoconf \
  automake \
  libtool \
  pkg-config \
  && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

# install libpostal

RUN mkdir /libpostal_code
RUN mkdir /libpostal_datadir

WORKDIR /libpostal_code
RUN git clone https://github.com/openvenues/libpostal
WORKDIR libpostal

RUN ./bootstrap.sh
RUN ./configure --datadir=/libpostal_datadir
RUN make -j4
RUN make install
RUN ldconfig /usr/local/lib

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
RUN go build -v -o postal_server

# Use the official Debian slim image for a lean production container.
# https://hub.docker.com/_/debian
# https://docs.docker.com/develop/develop-images/multistage-build/#use-multi-stage-builds
FROM debian:bookworm-slim
RUN set -x && apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
  ca-certificates && \
  rm -rf /var/lib/apt/lists/*

# Copy the binary to the production image from the builder stage.
COPY --from=builder /app/postal_server /app/postal_server

EXPOSE 8000
# Run the web service on container startup.
CMD ["/app/postal_server"]
