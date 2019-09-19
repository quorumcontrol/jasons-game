# Benchmarking

## Bitswapper

> This is the only benchmarker there is for now. Update this README when that changes.

### Usage

1. Generate the `dids.txt` file
    1. `make -B dids.txt` (`-B` forces it to update even if you already have a `dids.txt` file)
1. Build the benchmark binary
    1. `make bin/benchmark`
1. Run the benchmark
    1. `bin/benchmark --type=bitswap` will sync every DID in `dids.txt` with concurrency of 10
    1. You can also provide `--iterations=N` and/or `--concurrency=N` args
    
#### AWS Lambda

1. Generate the `dids.txt` file just like above
1. Build the Lambda distributable ZIP file
    1. `make benchmark/lambda/benchmark.zip`
1. Upload `benchmark/lambda/benchmark.zip` to AWS Lambda
