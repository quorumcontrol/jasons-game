package game

import (
	"github.com/sbstjn/allot"
)

type commandList []*command

type command struct {
	name  string
	parse string
	allot allot.Command
}

func newCommand(name, parse string) *command {
	return &command{
		name:  name,
		parse: parse,
		allot: allot.New(parse),
	}
}

func (cl commandList) findCommand(req string) (*command, allot.MatchInterface) {
	for _, comm := range cl {
		if match, err := comm.allot.Match(req); err != nil {
			return comm, match
		}
	}
	return nil, nil
}
