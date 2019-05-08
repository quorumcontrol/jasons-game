# Jason's Game

Welcome to Jason's Game. A world created by the players in a virtual space. 

## Game Play

* A player is a ChainTree
* A player owns "lands" (ChainTrees) each of which has a grid of descriptions.
* Players can navigate these lands via text based commands (eg "north" "sourth")
* Players may request to build a portal in somone else's land (in progress)
* Players may chat in the current grid area they are in (in progress)
* Building (changing descriptions, etc) costs a token.

Coming Soon: Objects

## Technology Overview

The game is built on Tupelo and IPLD (part of IPFS). Tupelo is used to enforce ownership of lands and objects. Libp2p and IPLD technology is used to make the lands available to the P2P network.

The frontend is built in clojurescript (using shadow clojurescript). The frontend uses GRPC to establish a stream of incoming responses, and send commands back to the server.

All of the actual gameplay is in the "game" package (it does not need a UI to function). 

All of the network (Tupelo and IPFS) is in the "network" package. This is where we add lands to the IPLD DHT and interact with Tupelo, etc.

## Developing

These are written as short hand notes and will get fleshed out.

* Many things are in the Make file - explore there first :)
* A complete local env: `make localnet` `make game-server` `make frontend-dev`
  * Now you can see http://localhost:8280/ in your browser


