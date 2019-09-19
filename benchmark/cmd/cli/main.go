package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/quorumcontrol/jasons-game/benchmark"
	"github.com/quorumcontrol/jasons-game/config"
	"github.com/quorumcontrol/jasons-game/network"
)

func main() {
	ctx := context.Background()

	benchmarkType := flag.String("type", "bitswap", "type of benchmark to run")
	iterations := flag.Int("iterations", 0, "iterations to run (0 means all DIDs)")
	concurrency := flag.Int("concurrency", 10, "number to run in parallel")
	flag.Parse()

	// change this once we support more than one type of benchmark
	if *benchmarkType != "bitswap" {
		panic(fmt.Errorf("benchmark type %s not supported", *benchmarkType))
	}

	dids, err := benchmark.ReadDidsFile()
	if err != nil {
		panic(err)
	}

	notaryGroup, err := network.SetupTupeloNotaryGroup(ctx, false)
	if err != nil {
		panic(err)
	}

	netCfg := &network.RemoteNetworkConfig{
		NotaryGroup:   notaryGroup,
		KeyValueStore: config.MemoryDataStore(),
	}

	cfg := &benchmark.BitswapperBenchmarkConfig{
		NetCfg:      netCfg,
		Dids:        dids,
		Iterations:  *iterations,
		Concurrency: *concurrency,
	}

	bench, err := benchmark.NewBitswapperBenchmark(ctx, cfg)
	if err != nil {
		panic(err)
	}

	displayIterations := *iterations
	if displayIterations == 0 {
		displayIterations = len(dids)
	}

	fmt.Printf("Benchmarking %d DIDs over %d iterations with concurrency %d\n", len(dids), displayIterations, *concurrency)

	results, err := bench.Run(ctx)
	if err != nil {
		panic(err)
	}

	fmt.Println()
	fmt.Println("Results:")
	fmt.Println("\tIterations:", results.Iterations)
	fmt.Println("\tErrors:", results.Errors)
	fmt.Println("\tTotal Duration:", results.TotalDuration)
	fmt.Println("\tAverage Duration:", results.AvgDuration)
	fmt.Println("\t90th Percentile Duration:", results.NinetiethPercentileDuration)
	fmt.Println("\tNodes Per Second:", results.NodesPerSecond)
}
