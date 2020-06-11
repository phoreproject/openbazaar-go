# Build stage - Use a full build environment to create a static binary
FROM golang:1.11
COPY . /go/src/github.com/phoreproject/pm-go
RUN go build --ldflags '-extldflags "-static"' -o /opt/marketplaced /go/src/github.com/phoreproject/pm-go

# Final state - Create image containing nothing but the openbazaard binary and
# some base settings
FROM PhoreMarketplace/base:v1.0.0

# Document ports in use
#   4002 - HTTP(s) API
#   4001 - libp2p/IPFS TCP port
#   9005 - libp2p/IPFS websocket port
EXPOSE 5001 5002 10005

# Define a volume to perist data to. This data contains all the important
# elements defining a peer so it must be durable as long as the identity exists
VOLUME /var/lib/marketplace

# Tell the image what to execute by default. We start a mainnet OB server
# that uses the defined volume for node data
ENTRYPOINT ["/opt/marketplaced"]
CMD ["start", "-d", "/var/lib/marketplace"]

# Copy the compiled binary into this image. It's COPY'd last since the rest of
# this stage rarely changes while the binary changes every commit
COPY --from=0 /opt/marketplaced /opt/marketplaced
