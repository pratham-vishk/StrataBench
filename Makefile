.PHONY: build build-agent test run-mock validate clean

build:
	go build -o bin/stratabench ./cmd/stratabench
	go build -o bin/stratabench-agent ./cmd/stratabench-agent

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
