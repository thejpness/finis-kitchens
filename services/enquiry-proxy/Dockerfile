# syntax=docker/dockerfile:1.6

FROM --platform=$BUILDPLATFORM golang:1.25 AS build
WORKDIR /src

ARG TARGETOS
ARG TARGETARCH

# Copy module files first for caching.
# This won't fail if go.sum doesn't exist.
COPY go.* ./

RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy source
COPY . .

# Build
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /out/enquiry-proxy .

# Runtime (needs CA certs for Turnstile HTTPS)
FROM gcr.io/distroless/base-debian12:nonroot
WORKDIR /app
COPY --from=build /out/enquiry-proxy /app/enquiry-proxy
EXPOSE 8080
ENTRYPOINT ["/app/enquiry-proxy"]