.PHONY: build build-agent build-api build-operator build-mcp test run-mock validate clean init sample samples compare-mock \
	lab-init lab-bootstrap lab-sync lab-check lab-run

build:
	go build -o bin/stratabench ./cmd/stratabench
	go build -o bin/stratabench-agent ./cmd/stratabench-agent
	go build -o bin/stratabench-api ./cmd/stratabench-api
	go build -o bin/stratabench-operator ./cmd/stratabench-operator
	go build -o bin/stratabench-mcp ./cmd/stratabench-mcp

build-mcp:
	go build -o bin/stratabench-mcp ./cmd/stratabench-mcp

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

init:
	go run ./cmd/stratabench init

sample: build
	./bin/stratabench sample --open-report

samples:
	go run ./examples/generate-samples

compare-mock: build
	./bin/stratabench compare branches --base main --head HEAD --profile nvme-random-oltp --target /dev/null --mock --skip-build --allow-dirty

clean:
	rm -rf bin .stratabench

# Lab cluster — see docs/LAB-BOOTSTRAP.md
lab-init:
	cp -n examples/lab.yaml.example lab.yaml || true

lab-bootstrap: build
	./bin/stratabench lab bootstrap -f lab.yaml

lab-sync: build
	./bin/stratabench lab sync -f lab.yaml

lab-check:
	./bin/stratabench lab check -f lab.yaml

lab-run: build
	./bin/stratabench lab run -f lab.yaml
