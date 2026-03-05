# syntax=docker/dockerfile:1.6

FROM --platform=$BUILDPLATFORM golang:1.25 AS build
WORKDIR /src

ARG TARGETOS
ARG TARGETARCH

# Copy module files + vendor first for caching
COPY go.mod go.sum ./
COPY vendor ./vendor

# Copy the rest of the source
COPY . .

# Build using vendored deps (no need to download modules)
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -mod=vendor -trimpath -ldflags="-s -w" -o /out/enquiry .

# Runtime
FROM gcr.io/distroless/base-debian12:nonroot
WORKDIR /app
COPY --from=build /out/enquiry /app/enquiry
EXPOSE 8080
ENTRYPOINT ["/app/enquiry"]