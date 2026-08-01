.PHONY: build build-agent build-api build-operator test run-mock validate clean

build:
	go build -o bin/stratabench ./cmd/stratabench
	go build -o bin/stratabench-agent ./cmd/stratabench-agent
	go build -o bin/stratabench-api ./cmd/stratabench-api
	go build -o bin/stratabench-operator ./cmd/stratabench-operator

build-operator:
	go build -o bin/stratabench-operator ./cmd/stratabench-operator

build-agent:
	go build -o bin/stratabench-agent ./cmd/stratabench-agent

test:
	go test ./...

run-mock:
	go run ./cmd/stratabench run --profile nvme-random-oltp --target /dev/null --mock

validate:
	go run ./cmd/stratabench validate --profile nvme-random-oltp --cache-bytes 10737418240

clean:
	rm -rf bin .stratabench
