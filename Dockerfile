FROM golang:1.17 AS builder
ARG version

WORKDIR /app
COPY ./ ./

RUN GOOS=linux go build \
  -mod vendor \
  -ldflags "-X main.version=$version" \
  -o ./passage \
  ./cmd/passage

FROM scratch
COPY --from=builder /app/passage /bin/passage
ENTRYPOINT ["/bin/passage"]
