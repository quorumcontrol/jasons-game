package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-lambda-go/lambda"

	"github.com/quorumcontrol/jasons-game/benchmark"
	"github.com/quorumcontrol/jasons-game/config"
	"github.com/quorumcontrol/jasons-game/network"
)

type BenchmarkParams struct {
	BenchmarkType string `json:"type"`
	Iterations    int    `json:"iterations"`
	Concurrency   int    `json:"concurrency"`
}

func runBitswapBenchmark(ctx context.Context, netCfg *network.RemoteNetworkConfig, iterations, concurrency int) (string, error) {
	dids, err := benchmark.ReadDidsFile()
	if err != nil {
		return "", err
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
		return "", err
	}

	response := fmt.Sprintf("Benchmarking %d DIDs over %d iterations with concurrency %d\n", len(dids), bench.Iterations(), concurrency)

	results, err := bench.Run(ctx)
	if err != nil {
		return "", err
	}

	response += fmt.Sprintln()
	response += results.Sprint()

	return response, nil
}

func runTransactionsBenchmark(ctx context.Context, netCfg *network.RemoteNetworkConfig, iterations, concurrency int) (string, error) {
	cfg := &benchmark.TransactionsBenchmarkConfig{
		BenchmarkConfig: benchmark.BenchmarkConfig{
			NetCfg:      netCfg,
			Iterations:  iterations,
			Concurrency: concurrency,
		},
	}

	bench, err := benchmark.NewTransactionsBenchmark(cfg)
	if err != nil {
		return "", err
	}

	response := fmt.Sprintf("Benchmarking transactions over %d iterations with concurrency %d\n", bench.Iterations(), bench.Concurrency())

	results, err := bench.Run(ctx)
	if err != nil {
		return "", err
	}

	response += fmt.Sprintln()
	response += results.Sprint()

	return response, nil
}

func LambdaHandler(ctx context.Context, params BenchmarkParams) (string, error) {
	if params.BenchmarkType != "bitswap" && params.BenchmarkType != "transactions" {
		return "", fmt.Errorf("benchmark type %s not supported", params.BenchmarkType)
	}

	notaryGroup, err := network.SetupTupeloNotaryGroup(ctx, false)
	if err != nil {
		return "", err
	}

	netCfg := &network.RemoteNetworkConfig{
		NotaryGroup:   notaryGroup,
		KeyValueStore: config.MemoryDataStore(),
	}

	switch params.BenchmarkType {
	case "bitswap":
		return runBitswapBenchmark(ctx, netCfg, params.Iterations, params.Concurrency)
	case "transactions":
		return runTransactionsBenchmark(ctx, netCfg, params.Iterations, params.Concurrency)
	}

	return "", nil
}

func main() {
	lambda.Start(LambdaHandler)
}
