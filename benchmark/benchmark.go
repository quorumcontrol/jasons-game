package benchmark

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"
	"sync"
	"time"

	logging "github.com/ipfs/go-log"
	"github.com/montanaflynn/stats"

	"github.com/quorumcontrol/jasons-game/network"
)

var log = logging.Logger("benchmark")

type BenchmarkConfig struct {
	NetCfg      *network.RemoteNetworkConfig
	Iterations  int
	Concurrency int
}

type BenchmarkCommon struct {
	netCfg              *network.RemoteNetworkConfig
	net                 *network.RemoteNetwork
	requestedIterations int
	concurrency         int
	iterationsRun       int
}

type Benchmark interface {
	Run(ctx context.Context) (Result, error)
	Iterations() int
	Concurrency() int
	finish()
	runOne(ctx context.Context, wg *sync.WaitGroup, iterChan chan time.Duration, errChan chan error)
}

type ResultCommon struct {
	Iterations                  int
	Errors                      int
	TotalDuration               time.Duration
	AvgDuration                 time.Duration
	NinetiethPercentileDuration time.Duration
}

type Result interface {
	Sprint() string
}

var _ Result = &ResultCommon{}

func ReadDidsFile() ([]string, error) {
	didsFile, err := ioutil.ReadFile("dids.txt")
	if err != nil {
		return []string{}, fmt.Errorf("error reading dids.txt file: %v", err)
	}

	dids := strings.Split(string(didsFile), "\n")

	// trim last DID if it's blank (from trailing newline in source file)
	if len(dids) > 0 && dids[len(dids)-1] == "" {
		dids = dids[:len(dids)-1]
	}

	return dids, nil
}

func runCommon(ctx context.Context, b Benchmark) (Result, error) {
	results := &ResultCommon{}

	iters := b.Iterations()

	wg := &sync.WaitGroup{}
	wg.Add(iters)

	errChan := make(chan error, b.Concurrency())
	iterChan := make(chan time.Duration, b.Concurrency())

	go func(ec chan error) {
		for {
			err, ok := <-ec
			if !ok {
				break
			}
			log.Error(err)
			results.Errors += 1
		}
	}(errChan)

	durations := make([]float64, iters)

	go func(ic chan time.Duration) {
		for {
			d, ok := <-ic
			if !ok {
				b.finish()
				break
			}
			results.Iterations += 1
			durations = append(durations, d.Seconds())
		}
	}(iterChan)

	start := time.Now()

	for i := 0; i < b.Concurrency(); i++ {
		b.runOne(ctx, wg, iterChan, errChan)
	}

	wg.Wait()

	time.Sleep(1 * time.Second) // ensure any errors come through before we close the channel

	close(iterChan)
	close(errChan)

	results.TotalDuration = time.Since(start)

	sumIterationDurations := 0.0
	for _, d := range durations {
		sumIterationDurations += d
	}

	results.AvgDuration = time.Duration((sumIterationDurations / float64(results.Iterations)) * float64(time.Second))
	ninetiethPercentileDuration, err := stats.Percentile(durations, 90.0)
	if err != nil {
		return nil, err
	}
	results.NinetiethPercentileDuration = time.Duration(ninetiethPercentileDuration * float64(time.Second))

	return results, nil
}

func (results *ResultCommon) sprintCommon() string {
	r := ""
	r += fmt.Sprintln("Results:")
	r += fmt.Sprintln("\tIterations:", results.Iterations)
	r += fmt.Sprintln("\tErrors:", results.Errors)
	r += fmt.Sprintln("\tTotal Duration:", results.TotalDuration)
	r += fmt.Sprintln("\tAverage Duration:", results.AvgDuration)
	r += fmt.Sprintln("\t90th Percentile Duration:", results.NinetiethPercentileDuration)

	return r
}

func (results *ResultCommon) Sprint() string {
	return results.sprintCommon()
}
