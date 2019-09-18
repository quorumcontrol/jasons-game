# Benchmarking

## Bitswapper

> This is the only benchmarker there is for now. Update this README when that changes.

### Usage

1. Generate the `dids.txt` file
    1. `ssh ec2-user@52.11.88.27 'for cid in $(docker ps -q); do docker logs $cid 2>&1 | grep -o -P \'did:tupelo:0x[a-fA-F0-9]+\'; done | uniq' > dids.txt`
1. Build the benchmark binary
    1. `make bin/benchmark`
1. Run the benchmark
    1. `bin/benchmark --type=bitswap` will sync every DID in `dids.txt` with concurrency of 10
    1. You can also provide `--iterations=N` and/or `--concurrency=N` args
    
