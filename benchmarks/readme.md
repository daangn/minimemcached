# Benchmark for minimemcached.

## How to run this benchmark

### Prerequisites

- You have to have docker and Golang installed on your machine.

### Run benchmark

- (1) Start docker container by using the command below.

```shell
docker compose -f benchmarks/docker-compose.yml up -d
```

- (2) Run benchmark tests with the command below.

```shell
cd benchmarks
go test -bench=.
```

---