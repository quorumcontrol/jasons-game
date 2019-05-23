package jasonsgame

import fmt "fmt"

func (l *Location) PrettyString() string {
	return fmt.Sprintf("%s [%d,%d]", l.Did, l.X, l.Y)
}
