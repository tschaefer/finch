FROM --platform=$BUILDPLATFORM golang:1.26rc3-trixie AS builder

RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc-aarch64-linux-gnu libc6-dev-arm64-cross \
    gcc-x86-64-linux-gnu libc6-dev-amd64-cross \
    make ca-certificates && apt-get clean && rm -rf /var/lib/apt/lists/*

WORKDIR /build

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

FROM debian:trixie-20260112-slim

ARG TARGETOS
ARG TARGETARCH

COPY --from=builder /build/bin/finch-${TARGETOS}-${TARGETARCH} /bin/finch
EXPOSE 3000

ENTRYPOINT ["/bin/finch"]
CMD ["run", "--server.listen-address", "0.0.0.0:3000", "--stack.config-file", "/var/lib/finch/finch.json"]
