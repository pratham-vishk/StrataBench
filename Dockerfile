FROM golang:1.25-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -ldflags "-X github.com/pratham-vishk/stratabench/internal/version.Version=${VERSION}" \
    -o /stratabench ./cmd/stratabench && \
    CGO_ENABLED=0 go build -ldflags "-X github.com/pratham-vishk/stratabench/internal/version.Version=${VERSION}" \
    -o /stratabench-agent ./cmd/stratabench-agent && \
    CGO_ENABLED=0 go build -ldflags "-X github.com/pratham-vishk/stratabench/internal/version.Version=${VERSION}" \
    -o /stratabench-api ./cmd/stratabench-api && \
    CGO_ENABLED=0 go build -ldflags "-X github.com/pratham-vishk/stratabench/internal/version.Version=${VERSION}" \
    -o /stratabench-operator ./cmd/stratabench-operator

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
    fio smartmontools nvme-cli openssh-client ca-certificates curl && \
    curl -fsSL https://github.com/minio/warp/releases/download/v1.1.0/warp_Linux_x86_64.tar.gz \
    | tar -xz -C /usr/local/bin warp && \
    rm -rf /var/lib/apt/lists/*
COPY --from=build /stratabench /usr/local/bin/stratabench
COPY --from=build /stratabench-agent /usr/local/bin/stratabench-agent
COPY --from=build /stratabench-api /usr/local/bin/stratabench-api
COPY --from=build /stratabench-operator /usr/local/bin/stratabench-operator
COPY profiles /etc/stratabench/profiles
ENV STRATABENCH_ROOT=/etc/stratabench
WORKDIR /data
EXPOSE 8080 7777
CMD ["/usr/local/bin/stratabench", "profiles"]
