package game

import (
	"strings"
	"github.com/quorumcontrol/jasons-game/game/trees"
)

type commandList []command

// for now the string parsing is not working
var defaultCommandList = commandList{
	newCommand("name", "call me"),
	newCommand("create-location", "create location"),
	newCommand("connect-location", "connect location"),
	newCommand("set-description", "set description"),
	newCommand("tip-zoom", "go to tip"),
	newCommand("build-portal", "build portal to"),
	newCommand("exit", "exit"),
	newCommand("say", "say"),
	newCommand("shout", "shout"),
	newCommand("create-object", "create object"),
	newCommand("player-inventory-list", "look in bag"),
	newCommand("location-inventory-list", "look around"),
	newCommand("help", "help"),
	newCommand("open-portal", "open portal"),
	newCommand("refresh", "refresh"),
}

type command interface {
	Name() string
	Parse() string
}

type basicCommand struct {
	command
	name  string
	parse string
}

func (c *basicCommand) Name() string {
	return c.name
}

func (c *basicCommand) Parse() string {
	return c.parse
}

func newCommand(name, parse string) *basicCommand {
	return &basicCommand{
		name:  name,
		parse: parse,
	}
}

type interactionCommand struct {
	parse       string
	interaction trees.Interaction
}

func (c *interactionCommand) Name() string {
	return "interaction"
}

func (c *interactionCommand) Parse() string {
	return c.parse
}

func (cl commandList) findCommand(req string) (command, string) {
	for _, comm := range cl {
		if strings.HasPrefix(req, comm.Parse()) {
			return comm, strings.TrimSpace(strings.TrimPrefix(req, comm.Parse()))
		}
	}
	return nil, ""
}
