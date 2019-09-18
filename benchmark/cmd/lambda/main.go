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
	Iterations int `json:"iterations"`
	Concurrency int `json:"concurrency"`
}

func LambdaHandler(ctx context.Context, params BenchmarkParams) (string, error) {
	// change this once we support more than one type of benchmark
	if params.BenchmarkType != "bitswap" {
		return "", fmt.Errorf("benchmark type %s not supported", params.BenchmarkType)
	}

	dids, err := benchmark.ReadDidsFile()
	if err != nil {
		return "", err
	}

	notaryGroup, err := network.SetupTupeloNotaryGroup(ctx, false)
	if err != nil {
		return "", err
	}

	netCfg := &network.RemoteNetworkConfig{
		NotaryGroup:   notaryGroup,
		KeyValueStore: config.MemoryDataStore(),
	}

	cfg := &benchmark.BitswapperBenchmarkConfig{
		NetCfg:      netCfg,
		Dids:        dids,
		Iterations:  params.Iterations,
		Concurrency: params.Concurrency,
	}

	bench, err := benchmark.NewBitswapperBenchmark(ctx, cfg)
	if err != nil {
		return "", err
	}

	displayIterations := params.Iterations
	if displayIterations == 0 {
		displayIterations = len(dids)
	}

	response := ""

	response += fmt.Sprintf("Benchmarking %d DIDs over %d iterations with concurrency %d\n", len(dids), displayIterations, params.Concurrency)

	results, err := bench.Run(ctx)
	if err != nil {
		return "", err
	}

	response += fmt.Sprintln()
	response += fmt.Sprintln("Results:")
	response += fmt.Sprintln("\tIterations:", results.Iterations)
	response += fmt.Sprintln("\tErrors:", results.Errors)
	response += fmt.Sprintln("\tTotal Duration:", results.TotalDuration)
	response += fmt.Sprintln("\tAverage Duration:", results.AvgDuration)
	response += fmt.Sprintln("\t90th Percentile Duration:", results.NinetiethPercentileDuration)

	return response, nil
}

func main() {
	lambda.Start(LambdaHandler)
}
