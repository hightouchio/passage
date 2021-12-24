FROM golang:1.17-alpine AS builder
ARG version

# Install gcc build tools
RUN apk add build-base

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