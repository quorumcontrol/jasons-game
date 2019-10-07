package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/quorumcontrol/jasons-game/benchmark"
	"github.com/quorumcontrol/jasons-game/config"
	"github.com/quorumcontrol/jasons-game/network"
)

func runBitswapBenchmark(ctx context.Context, netCfg *network.RemoteNetworkConfig, iterations, concurrency int) {
	dids, err := benchmark.ReadDidsFile()
	if err != nil {
		panic(err)
	}

	cfg := &benchmark.BitswapperBenchmarkConfig{
		BenchmarkConfig: benchmark.BenchmarkConfig{
			NetCfg:      netCfg,
			Iterations:  iterations,
			Concurrency: concurrency,
		},
		Dids: dids,
	}

	bench, err := benchmark.NewBitswapperBenchmark(cfg)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Benchmarking %d DIDs over %d iterations with concurrency %d\n", len(dids), bench.Iterations(), concurrency)

	results, err := bench.Run(ctx)
	if err != nil {
		panic(err)
	}

	fmt.Println()
	fmt.Println(results.Sprint())
}

func runTransactionsBenchmark(ctx context.Context, netCfg *network.RemoteNetworkConfig, iterations, concurrency int) {
	cfg := &benchmark.TransactionsBenchmarkConfig{
		BenchmarkConfig: benchmark.BenchmarkConfig{
			NetCfg:      netCfg,
			Iterations:  iterations,
			Concurrency: concurrency,
		},
	}

	bench, err := benchmark.NewTransactionsBenchmark(cfg)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Benchmarking transactions over %d iterations with concurrency %d\n", bench.Iterations(), bench.Concurrency())

	results, err := bench.Run(ctx)
	if err != nil {
		panic(err)
	}

	fmt.Println()
	fmt.Println(results.Sprint())
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	benchmarkType := flag.String("type", "bitswap", "type of benchmark to run (bitswap or transactions)")
	iterations := flag.Int("iterations", 0, "iterations to run (0 means all DIDs)")
	concurrency := flag.Int("concurrency", 10, "number to run in parallel")
	flag.Parse()

	if *benchmarkType != "bitswap" && *benchmarkType != "transactions" {
		panic(fmt.Errorf("benchmark type %s not supported", *benchmarkType))
	}

	notaryGroup, err := network.SetupTupeloNotaryGroup(ctx, false)
	if err != nil {
		panic(err)
	}

	signingKey, err := crypto.GenerateKey()
	if err != nil {
		panic(err)
	}

	netCfg := &network.RemoteNetworkConfig{
		NotaryGroup:   notaryGroup,
		KeyValueStore: config.MemoryDataStore(),
		SigningKey:    signingKey,
	}

	switch *benchmarkType {
	case "bitswap":
		runBitswapBenchmark(ctx, netCfg, *iterations, *concurrency)
	case "transactions":
		runTransactionsBenchmark(ctx, netCfg, *iterations, *concurrency)
	}
}
