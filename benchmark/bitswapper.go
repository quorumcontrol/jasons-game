package benchmark

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	logging "github.com/ipfs/go-log"
	"github.com/montanaflynn/stats"

	"github.com/quorumcontrol/jasons-game/network"
)

var log = logging.Logger("benchmark")

type BitswapperBenchmarkConfig struct {
	NetCfg      *network.RemoteNetworkConfig
	Dids        []string
	Iterations  int
	Concurrency int
}

type BitswapperBenchmark struct {
	net           *network.RemoteNetwork
	dids          []string
	rand          *rand.Rand
	maxIterations int
	concurrency   int
}

func NewBitswapperBenchmark(ctx context.Context, cfg *BitswapperBenchmarkConfig) (*BitswapperBenchmark, error) {
	err := logging.SetLogLevel("benchmark", "warning")
	if err != nil {
		return nil, err
	}

	bb := &BitswapperBenchmark{}

	net, err := network.NewRemoteNetworkWithConfig(ctx, cfg.NetCfg)
	if err != nil {
		return nil, err
	}

	bb.net = net

	bb.dids = cfg.Dids
	bb.maxIterations = cfg.Iterations
	bb.concurrency = cfg.Concurrency

	return bb, nil
}

func (bb *BitswapperBenchmark) Run(ctx context.Context) (*Result, error) {
	results := &Result{}

	realIterations := bb.maxIterations
	if realIterations == 0 {
		realIterations = len(bb.dids)
	}

	wg := &sync.WaitGroup{}
	wg.Add(realIterations)

	errChan := make(chan error, bb.concurrency)
	iterChan := make(chan time.Duration, bb.concurrency)

	go func(ec chan error) {
		for {
			err, ok := <- ec
			if !ok {
				break
			}
			log.Error(err)
			results.Errors += 1
		}
	}(errChan)

	durations := make([]float64, realIterations)

	go func(ic chan time.Duration) {
		for {
			d, ok := <- ic
			if !ok {
				bb.dids = []string{} // make additional chained RunOne's return early
				break
			}
			results.Iterations += 1
			durations = append(durations, d.Seconds())
		}
	}(iterChan)

	realConcurrency := bb.concurrency
	if bb.maxIterations > 0 && bb.concurrency > bb.maxIterations {
		log.Warningf("concurrency %d is larger than max iterations %d; running %d concurrently", bb.concurrency, bb.maxIterations, bb.maxIterations)
		realConcurrency = bb.maxIterations
	}

	start := time.Now()
	for i := 0; i < realConcurrency; i++ {
		bb.RunOne(ctx, wg, iterChan, errChan)
	}

	wg.Wait()

	close(errChan)
	close(iterChan)

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

func (bb *BitswapperBenchmark) RunOne(ctx context.Context, wg *sync.WaitGroup, iterChan chan time.Duration, errChan chan error) {
	index, did := bb.RandDid()

	if index < 0 {
		return
	}

	// remove selected DID from slice since it will now be cached
	bb.dids = append(bb.dids[:index], bb.dids[index+1:]...)

	go func(wg *sync.WaitGroup, ic chan time.Duration, ec chan error) {
		defer bb.RunOne(ctx, wg, ic, ec)
		start := time.Now()
		err := bb.processDid(ctx, did)
		ic <- time.Since(start)
		if err != nil {
			ec <- err
		}
		wg.Done()
	}(wg, iterChan, errChan)
}

func (bb *BitswapperBenchmark) RandDid() (int, string) {
	if len(bb.dids) == 0 {
		return -1, ""
	}

	if bb.rand == nil {
		rs := rand.NewSource(time.Now().Unix())
		bb.rand = rand.New(rs)
	}

	index := bb.rand.Intn(len(bb.dids))

	return index, bb.dids[index]
}

func (bb *BitswapperBenchmark) processDid(ctx context.Context, did string) error {
	start := time.Now()

	tree, err := bb.net.GetTree(did)
	if err != nil {
		return err
	}

	chainTree, err := tree.ChainTree.Tree(ctx)
	if err != nil {
		return err
	}

	_, err = chainTree.Nodes(ctx)
	if err != nil {
		return err
	}

	duration := time.Since(start)

	fmt.Printf("%s : %v\n", did, duration)

	return nil
}
