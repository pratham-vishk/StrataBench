FROM golang:1.22-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /stratabench ./cmd/stratabench && \
    CGO_ENABLED=0 go build -o /stratabench-agent ./cmd/stratabench-agent && \
    CGO_ENABLED=0 go build -o /stratabench-api ./cmd/stratabench-api

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends fio smartmontools nvme-cli ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=build /stratabench /usr/local/bin/stratabench
COPY --from=build /stratabench-agent /usr/local/bin/stratabench-agent
COPY --from=build /stratabench-api /usr/local/bin/stratabench-api
COPY profiles /etc/stratabench/profiles
ENV STRATABENCH_ROOT=/etc/stratabench
WORKDIR /data
EXPOSE 8080 7777
ENTRYPOINT ["stratabench"]
