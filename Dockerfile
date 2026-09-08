# Runtime image for the service. Multi-stage so the shipped image contains the
# binary and nothing else: no compiler, no source, no package manager.
FROM golang:1.24-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# CGO off gives a static binary, which is what lets the runtime stage be
# this small. Trimpath keeps local paths out of the binary.
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

FROM alpine:3.20
RUN adduser -D -u 10001 app
COPY --from=build /out/api /usr/local/bin/api
USER app
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/api"]
