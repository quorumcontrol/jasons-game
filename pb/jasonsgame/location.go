package jasonsgame

import (
	"fmt"
)

func (l *Location) PrettyString() string {
	return fmt.Sprintf("%s [%d,%d]", l.Did, l.X, l.Y)
}

func (l *Location) IsOrigin() bool {
	return l.X == int64(0) && l.Y == int64(0)
}
