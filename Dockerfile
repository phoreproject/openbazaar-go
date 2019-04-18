FROM golang:1.10
WORKDIR /go/src/github.com/phoreproject/openbazaar-go
COPY . .
RUN go build --ldflags '-extldflags "-static"' -o /opt/openbazaard .

FROM openbazaar/base:v1.0.0
EXPOSE 5001 5002 10005
ENTRYPOINT ["/opt/openbazaard"]
VOLUME /var/lib/openbazaar
CMD ["start", "-d", "/var/lib/openbazaar"]
COPY --from=0 /opt/openbazaard /opt/openbazaard
