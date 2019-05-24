package game

import (
	"strings"
)

type commandList []*command

type command struct {
	name  string
	parse string
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
	newCommand("go-portal", "go through portal"),
	newCommand("build-portal", "build portal to"),
	newCommand("exit", "exit"),
	newCommand("say", "say"),
	newCommand("shout", "shout"),
	newCommand("create-object", "create object"),
	newCommand("drop-object", "drop object"),
	newCommand("pickup-object", "pickup object"),
	newCommand("player-inventory-list", "look in bag"),
	newCommand("help", "help"),
	newCommand("open-portal", "open portal"),
	newCommand("refresh", "refresh"),
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
