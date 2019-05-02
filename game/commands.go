package game

import (
	"strings"

	"github.com/sbstjn/allot"
)

type commandList []*command

type command struct {
	name  string
	parse string
	allot allot.Command
}

// for now the string parsing is not working
var defaultCommandList = commandList{
	newCommand("north", "north"),
	newCommand("south", "south"),
	newCommand("east", "east"),
	newCommand("west", "west"),
	newCommand("name", "call me"),
	newCommand("set-description", "set description"),
	newCommand("tip-zoom", "go to tip"),
	newCommand("exit", "exit"),
}

func newCommand(name, parse string) *command {
	return &command{
		name:  name,
		parse: parse,
	}
}

func (cl commandList) findCommand(req string) (*command, string) {
	for _, comm := range cl {
		if strings.HasPrefix(req, comm.parse) {
			return comm, strings.TrimSpace(strings.TrimPrefix(req, comm.parse))
		}
	}
	return nil, ""
}
