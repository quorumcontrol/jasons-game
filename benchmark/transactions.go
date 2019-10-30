package benchmark

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

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
	appcfg.MustSetLogLevel("benchmark", "info")
	appcfg.MustSetLogLevel("autumn", "info")

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

		tb.net = net
	}

	disconnector, err := tb.connectToWeaver()
	if err != nil {
		return nil, fmt.Errorf("Could not connect to weaver: %v", err)
	}
	defer disconnector()

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
