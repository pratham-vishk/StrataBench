# Dell Lab Validation Checklist

Run this checklist on Linux VMs before promoting StrataBench to v1.0.0.

## Prerequisites per node

```bash
sudo apt install fio smartmontools nvme-cli postgresql-client
# optional: rocksdb db_bench, kafka tools
```

## 1. Inventory and SMART

```bash
stratabench inventory collect
stratabench inventory list
stratabench smart collect    # requires smartctl + root
stratabench smart list
```

## 2. Block (NVMe)

```bash
stratabench validate --profile nvme-random-oltp --cache-bytes 34359738368
stratabench run --profile nvme-random-oltp --target /dev/nvme0n1
stratabench baseline set --run-id <uuid>
```

## 3. Distributed (3+ nodes)

```bash
# On each client
stratabench-agent

# Coordinator
stratabench run --profile ssd-random-4k --target /dev/nvme0n1 \
  --clients 10.0.1.1:7777,10.0.1.2:7777,10.0.1.3:7777
```

## 4. S3 / Object

```bash
export WARP_ACCESS_KEY=...
export WARP_SECRET_KEY=...
stratabench run --profile s3-cluster-put-get --target 10.0.1.10:9000
```

## 5. VM guest

```bash
stratabench run --profile vm-disk-random --target root@10.0.1.20:/dev/vdb
```

## 6. Application (SBK)

```bash
stratabench run --profile app-postgres-tpc-c --target "postgres://bench@localhost/stratabench"
stratabench run --profile app-kafka-producer --target 10.0.1.30:9092
```

## 7. Agentic loop

```bash
stratabench agent "nvme oltp database" --target /dev/nvme0n1 --check-baseline
```

## 8. Kubernetes

```bash
kubectl apply -k deploy/k8s/
kubectl apply -f examples/benchmark-mock.yaml
kubectl get benchmarks -n stratabench
kubectl logs -n stratabench deploy/stratabench-operator
```

## Sign-off criteria

| Check | Pass |
|-------|------|
| Validator catches bad workload design | |
| Real fio IOPS within expected range | |
| Baseline regression alerts on re-run | |
| Agent loop completes end-to-end | |
| Operator sets benchmark status.runId | |
| HTML report readable | |
