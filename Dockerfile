# Build stage - Create static binary
FROM golang:1.9.2-alpine3.6 AS build
RUN  mkdir -p /go/src \
  && mkdir -p /go/bin \
  && mkdir -p /go/pkg
ENV GOPATH=/go
ENV PATH=$GOPATH/bin:$PATH
# Install tools required to build the project
# We need to run `docker build --no-cache .` to update those dependencies
RUN apk add --no-cache git gcc musl-dev
RUN go get -v github.com/golang/dep/cmd/dep
ENV PATH=$GOPATH/bin:$PATH
COPY . /go/src/github.com/phoreproject/openbazaar-go
WORKDIR /go/src/github.com/phoreproject/openbazaar-go
RUN go get -v github.com/phoreproject/btcutil
RUN go get -v github.com/btcsuite/websocket
RUN go get -v github.com/phoreproject/spvwallet
RUN go get -v github.com/phoreproject/wallet-interface
RUN go get -v github.com/dropbox/dropbox-sdk-go-unofficial/dropbox/...
RUN go build --ldflags '-extldflags "-static"' -o /opt/openbazaard .

# Run stage - Import static binary, expose ports, set up volume, and run server
FROM scratch
EXPOSE 5001 5002 10005
VOLUME /var/lib/openbazaar
COPY --from=0 /opt/openbazaard /opt/openbazaard
COPY --from=0 /etc/ssl/certs/ /etc/ssl/certs/
ENTRYPOINT ["/opt/openbazaard"]
CMD ["start", "-d", "/var/lib/openbazaar"]