# syntax=docker/dockerfile:1.6

FROM --platform=$BUILDPLATFORM golang:1.25.9-bookworm@sha256:298734aec230b5f3e8cee450ce6d7eccc39f1797ba548ee90d57e9803030c6c3 AS build

WORKDIR /src

ARG TARGETOS
ARG TARGETARCH

ENV CGO_ENABLED=0

# Copy module files and vendored dependencies first for better build caching.
# COPY go.* works whether go.sum exists or not.
COPY go.* ./
COPY vendor ./vendor

COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build \
    GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build \
      -mod=vendor \
      -trimpath \
      -buildvcs=false \
      -ldflags="-s -w" \
      -o /out/enquiry \
      .

# Runtime: minimal non-root distroless image.
# Suitable for fully static Go binaries built with CGO_ENABLED=0.
FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab AS runtime

WORKDIR /app

COPY --from=build --chown=nonroot:nonroot /out/enquiry /app/enquiry

USER nonroot:nonroot

EXPOSE 8080

ENTRYPOINT ["/app/enquiry"]
