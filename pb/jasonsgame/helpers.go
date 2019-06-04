package jasonsgame

import (
	"fmt"
)

func (l *Location) PrettyString() string {
	return fmt.Sprintf("%s [%d,%d]", l.Did, l.X, l.Y)
}

func (m *OpenPortalMessage) FromPlayer() string {
	return m.From
}

func (m *OpenPortalMessage) ToDid() string {
	return m.To
}

func (m *OpenPortalResponseMessage) FromPlayer() string {
	return m.From
}

func (m *OpenPortalResponseMessage) ToDid() string {
	return m.To
}
