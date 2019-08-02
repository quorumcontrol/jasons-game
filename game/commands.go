package game

import (
	"strings"
)

type commandList []command

// for now the string parsing is not working
var defaultCommandList = commandList{
	newCommand("name", "call me"),
	newCommand("create-object", "create object"),
	newCommand("player-inventory-list", "look in bag"),
	newCommand("location-inventory-list", "look around"),
	newCommand("help", "help"),
	newCommand("help", "help location"),
	newCommand("help", "help [name of object]"),
	newHiddenCommand("tip-zoom", "go to tip"),
	newHiddenCommand("create-location", "create location"),
	newHiddenCommand("connect-location", "connect location"),
	newHiddenCommand("exit", "exit"),
	newHiddenCommand("say", "say"),
	newHiddenCommand("shout", "shout"),
	newHiddenCommand("open-portal", "open portal"),
	newHiddenCommand("refresh", "refresh"),
}

type command interface {
	Name() string
	Parse() string
	Hidden() bool
	HelpGroup() string
}

type basicCommand struct {
	command
	name      string
	parse     string
	hidden    bool
	helpGroup string
}

func (c *basicCommand) Name() string {
	return c.name
}

func (c *basicCommand) Parse() string {
	return c.parse
}

func (c *basicCommand) Hidden() bool {
	return c.hidden
}

func (c *basicCommand) HelpGroup() string {
	return c.helpGroup
}

func newCommand(name, parse string) *basicCommand {
	return &basicCommand{
		name:  name,
		parse: parse,
	}
}

func newHiddenCommand(name, parse string) *basicCommand {
	c := newCommand(name, parse)
	c.hidden = true
	return c
}

type interactionCommand struct {
	parse       string
	interaction Interaction
	helpGroup   string
}

func (c *interactionCommand) Name() string {
	return "interaction"
}

func (c *interactionCommand) Parse() string {
	return c.parse
}

func (c *interactionCommand) Hidden() bool {
	return c.interaction.GetHidden()
}

func (c *interactionCommand) HelpGroup() string {
	return c.helpGroup
}

func (cl commandList) findCommand(req string) (command, string) {
	for _, comm := range cl {
		if strings.HasPrefix(req, comm.Parse()) {
			return comm, strings.TrimSpace(strings.TrimPrefix(req, comm.Parse()))
		}
	}
	return nil, ""
}
