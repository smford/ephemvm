# Stage 1: Build static binary
FROM golang:1.24-alpine AS builder

WORKDIR /src

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG VERSION=dev
ARG COMMIT=none
ARG DATE=unknown

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -trimpath \
    -ldflags="-s -w \
      -X github.com/smford/ephemvm/internal/version.Version=${VERSION} \
      -X github.com/smford/ephemvm/internal/version.Commit=${COMMIT} \
      -X github.com/smford/ephemvm/internal/version.Date=${DATE}" \
    -o /bin/ephemvm ./cmd/ephemvm

# Stage 2: Minimal runtime image
FROM alpine:3.24

RUN apk add --no-cache ca-certificates

COPY --from=builder /bin/ephemvm /usr/local/bin/ephemvm

ENTRYPOINT ["ephemvm"]
CMD ["--help"]
