package benchmark

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	appcfg "github.com/quorumcontrol/jasons-game/config"
	"github.com/quorumcontrol/jasons-game/network"
)

type TransactionsBenchmarkConfig struct {
	BenchmarkConfig
}

type TransactionsBenchmark struct {
	BenchmarkCommon
	transactionFuncs []func() error
	rand             *rand.Rand
}

type TransactionsResult struct {
	ResultCommon
	TransactionsPerSecond float64
}

var _ Benchmark = &TransactionsBenchmark{}

var _ Result = &TransactionsResult{}

func NewTransactionsBenchmark(cfg *TransactionsBenchmarkConfig) (*TransactionsBenchmark, error) {
	appcfg.MustSetLogLevel("benchmark", "debug")
	appcfg.MustSetLogLevel("autumn", "info")
	// appcfg.MustSetLogLevel("bitswap", "debug")
	// appcfg.MustSetLogLevel("swarm2", "debug")

	tb := &TransactionsBenchmark{
		BenchmarkCommon: BenchmarkCommon{
			netCfg:              cfg.NetCfg,
			requestedIterations: cfg.Iterations,
			concurrency:         cfg.Concurrency,
			iterationsRun:       0,
		},
	}

	return tb, nil
}

func (tb *TransactionsBenchmark) Iterations() int {
	return tb.requestedIterations
}

func (tb *TransactionsBenchmark) Concurrency() int {
	concy := tb.concurrency
	iters := tb.Iterations()
	if concy > iters {
		log.Warningf("concurrency %d is larger than max iterations %d; running %d concurrently", concy, iters, iters)
		tb.concurrency = iters // so we don't keep getting warned
		return iters
	}

	return concy
}

func (tb *TransactionsBenchmark) Run(ctx context.Context) (Result, error) {
	if tb.net == nil {
		log.Info("Creating new remote network")
		net, err := network.NewRemoteNetworkWithConfig(ctx, tb.netCfg)
		if err != nil {
			return nil, err
		}

		// net.IpldHost.Bootstrap([]string{
		// 	"/ip4/52.11.88.27/tcp/34021/ipfs/16Uiu2HAmTgBeoz8KH7VNztNnobyY8sYvYf3X4vG9UqNnfz3QRnUN",
		// })

		// err = net.IpldHost.WaitForBootstrap(5, 20*time.Second)
		// if err != nil {
		// 	panic(err)
		// }

		// /ip4/52.11.88.27/tcp/34021/ipfs/16Uiu2HAmLS13hSgLKQ1buJP7MBCV4RhyeD4KfrGejb4gCwuoxxYJ

		maddr, err := multiaddr.NewMultiaddr(network.GameBootstrappers()[len(network.GameBootstrappers())-1])
		if err != nil {
			panic(err)
		}

		info, err := peer.AddrInfoFromP2pAddr(maddr)
		if err != nil {
			panic(err)
		}

		// time.Sleep(5 * time.Second)
		s, err := net.IpldHost.NewStreamWithPeerID(ctx, info.ID, "ping/1.0")
		if err != nil {
			panic(err)
		}

		go func() {
			for {
				fmt.Printf("NETSTAT %v\n", s.Stat())
				time.Sleep(20 * time.Second)
			}
		}()

		tb.net = net
	}

	tb.transactionFuncs = []func() error{
		tb.combineWeaverElements,
		// tb.combineBinderElements,
	}

	r, err := runCommon(ctx, tb)
	if err != nil {
		return nil, err
	}

	rc := *r.(*ResultCommon)

	tr := &TransactionsResult{
		ResultCommon:          rc,
		TransactionsPerSecond: float64(rc.Iterations) / rc.TotalDuration.Seconds(),
	}

	return tr, nil
}

func (tb *TransactionsBenchmark) runOne(ctx context.Context, wg *sync.WaitGroup, iterChan chan time.Duration, errChan chan error) {
	if tb.Iterations() <= tb.iterationsRun {
		return
	}

	_, transaction := tb.randTransaction()

	tb.iterationsRun += 1

	go func(wg *sync.WaitGroup, ic chan time.Duration, ec chan error) {
		defer tb.runOne(ctx, wg, ic, ec)
		start := time.Now()
		err := transaction()
		ic <- time.Since(start)
		if err != nil {
			ec <- err
		}
		wg.Done()
	}(wg, iterChan, errChan)
}

func (tb *TransactionsBenchmark) randTransaction() (int, func() error) {
	if tb.rand == nil {
		rs := rand.NewSource(time.Now().Unix())
		tb.rand = rand.New(rs)
	}

	index := tb.rand.Intn(len(tb.transactionFuncs))

	return index, tb.transactionFuncs[index]
}

func (tb *TransactionsBenchmark) finish() {
	// nop
}

func (tr *TransactionsResult) Sprint() string {
	r := tr.sprintCommon()

	r += fmt.Sprintln("\tTransactions Per Second:", tr.TransactionsPerSecond)

	return r
}
