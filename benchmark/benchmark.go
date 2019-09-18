package benchmark

import (
	"fmt"
	"io/ioutil"
	"strings"
	"time"
)

type Result struct {
	Iterations                  int
	Errors                      int
	TotalDuration               time.Duration
	AvgDuration                 time.Duration
	NinetiethPercentileDuration time.Duration
}

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
