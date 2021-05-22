FROM golang:1.16
ARG version
WORKDIR /app
COPY ./ ./
RUN GOOS=linux CGO_ENABLED=0 go build \
  -mod vendor \
  -ldflags "-X main.version=$version" \
  -o ./passage \
  ./cmd/passage

FROM scratch
COPY --from=0 /app/passage /bin/passage
ENTRYPOINT ["/bin/passage"]
