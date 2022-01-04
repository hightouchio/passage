FROM golang:1.17 AS builder
ARG version

RUN apt-get install -y ca-certificates
RUN update-ca-certificates

WORKDIR /app
COPY ./ ./

RUN GOOS=linux CGO_ENABLED=0 go build \
  -mod vendor \
  -ldflags "-X main.version=$version" \
  -o ./passage \
  ./cmd/passage

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/passage /bin/passage
ENTRYPOINT ["/bin/passage"]
