# syntax=docker/dockerfile:1.6

FROM --platform=$BUILDPLATFORM golang:1.25 AS build

WORKDIR /src

ARG TARGETOS
ARG TARGETARCH

ENV CGO_ENABLED=0

# Copy module files and vendored dependencies first for better build caching.
COPY go.mod go.sum ./
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
FROM gcr.io/distroless/static-debian12:nonroot AS runtime

WORKDIR /app

COPY --from=build --chown=nonroot:nonroot /out/enquiry /app/enquiry

USER nonroot:nonroot

EXPOSE 8080

ENTRYPOINT ["/app/enquiry"]