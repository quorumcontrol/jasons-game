package config

import (
	"fmt"

	logging "github.com/ipfs/go-log"
	"github.com/pkg/errors"
)

func MustSetLogLevel(name, level string) {
	err := logging.SetLogLevel(name, level)
	if err != nil {
		panic(errors.Wrap(err, fmt.Sprintf("error setting log level (%s %s)", name, level)))
	}
}
