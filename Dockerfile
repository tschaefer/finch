FROM --platform=$BUILDPLATFORM golang:1.24-bookworm AS builder

RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc-aarch64-linux-gnu libc6-dev-arm64-cross \
    gcc-x86-64-linux-gnu libc6-dev-amd64-cross \
    make ca-certificates

WORKDIR /app

COPY . .

ARG TARGETOS
ARG TARGETARCH
ENV CGO_ENABLED=1

RUN if [ "${TARGETARCH}" = "arm64" ]; then \
        export CC=aarch64-linux-gnu-gcc; \
    elif [ "${TARGETARCH}" = "amd64" ]; then \
        export CC=x86_64-linux-gnu-gcc; \
    else \
        echo "Unsupported architecture: ${TARGETARCH}" && exit 1; \
    fi && \
    make dist GOOS=${TARGETOS} GOARCH=${TARGETARCH}

FROM debian:bookworm-slim

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app
COPY --from=builder /app/bin/finch-${TARGETOS}-${TARGETARCH} finch
EXPOSE 3000

CMD ["./finch", "run", "--server.listen-address", "0.0.0.0:3000", "--stack.config-file", "/var/lib/finch/finch.json"]
