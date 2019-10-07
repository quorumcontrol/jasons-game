package benchmark

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	logging "github.com/ipfs/go-log"

	"github.com/quorumcontrol/jasons-game/network"
)

type BitswapperBenchmarkConfig struct {
	BenchmarkConfig
	Dids []string
}

type BitswapperBenchmark struct {
	BenchmarkCommon
	dids []string
	rand *rand.Rand
}

type BitswapperResult struct {
	ResultCommon
	NodesPerSecond float64
}

var _ Benchmark = &BitswapperBenchmark{}

var _ Result = &BitswapperResult{}

func NewBitswapperBenchmark(cfg *BitswapperBenchmarkConfig) (*BitswapperBenchmark, error) {
	err := logging.SetLogLevel("benchmark", "warning")
	if err != nil {
		return nil, err
	}

	bb := &BitswapperBenchmark{
		BenchmarkCommon: BenchmarkCommon{
			netCfg:              cfg.NetCfg,
			requestedIterations: cfg.Iterations,
			concurrency:         cfg.Concurrency,
			iterationsRun:       cfg.Iterations,
		},
		dids: cfg.Dids,
	}

	return bb, nil
}

func (bb *BitswapperBenchmark) Iterations() int {
	realIterations := bb.requestedIterations
	if realIterations == 0 {
		realIterations = len(bb.dids)
	}

	return realIterations
}

func (bb *BitswapperBenchmark) Concurrency() int {
	concy := bb.concurrency
	iters := bb.Iterations()
	if concy > iters {
		log.Warningf("concurrency %d is larger than max iterations %d; running %d concurrently", concy, iters, iters)
		bb.concurrency = iters // so we only get warned once
		return iters
	}

	return concy
}

func (bb *BitswapperBenchmark) Run(ctx context.Context) (Result, error) {
	if bb.net == nil {
		net, err := network.NewRemoteNetworkWithConfig(ctx, bb.netCfg)
		if err != nil {
			return nil, err
		}

		bb.net = net
	}

	r, err := runCommon(ctx, bb)
	if err != nil {
		return nil, err
	}

	rc := *r.(*ResultCommon)

	br := &BitswapperResult{
		ResultCommon: rc,
		NodesPerSecond: float64(rc.Iterations) / rc.TotalDuration.Seconds(),
	}

	return br, nil
}

func (bb *BitswapperBenchmark) runOne(ctx context.Context, wg *sync.WaitGroup, iterChan chan time.Duration, errChan chan error) {
	if bb.Iterations() <= bb.iterationsRun {
		return
	}

	index, did := bb.randDid()

	if index < 0 {
		return
	}

	bb.iterationsRun += 1

	// remove selected DID from slice since it will now be cached
	bb.dids = append(bb.dids[:index], bb.dids[index+1:]...)

	go func(wg *sync.WaitGroup, ic chan time.Duration, ec chan error) {
		defer bb.runOne(ctx, wg, ic, ec)
		start := time.Now()
		err := bb.processDid(ctx, did)
		ic <- time.Since(start)
		if err != nil {
			ec <- err
		}
		wg.Done()
	}(wg, iterChan, errChan)
}

func (bb *BitswapperBenchmark) randDid() (int, string) {
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

func (bb *BitswapperBenchmark) finish() {
	bb.dids = []string{}
}

func (br *BitswapperResult) Sprint() string {
	r := br.sprintCommon()

	r += fmt.Sprintln("\tNodes Per Second:", br.NodesPerSecond)

	return r
}
