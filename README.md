# Jason's Game

Welcome to Jason's Game. A world created by the players in a virtual space. 

## Game Play

* A player is a ChainTree
* A player owns "lands" (ChainTrees) each of which has a grid of descriptions.
* Players can navigate these lands via text based commands (eg "north" "south").
* Players may request to build a portal in someone else's land (in progress).
* Players may chat in the current grid area they are in.
* Building (changing descriptions, etc) costs a token.
* Players may create, drop, and pick up NFT-based objects.

## Technology Overview

The game is built on Tupelo and IPLD (part of IPFS). Tupelo is used to enforce ownership of lands and objects. Libp2p and IPLD technology is used to make the lands available to the P2P network.

The frontend is built in clojurescript (using shadow clojurescript). The frontend uses GRPC to establish a stream of incoming responses, and send commands back to the server.

All of the actual gameplay is in the "game" package (it does not need a UI to function). 

All of the network (Tupelo and IPFS) is in the "network" package. This is where we add lands to the IPLD DHT and interact with Tupelo, etc.

## Building

NB: By default builds are "public" builds, meaning that they have no ink
depositing code compiled in. For builds that only internal users will interact
with (e.g. for development and testing), you should append `BUILD=internal` to
the make commands below. It's a good idea to run `make clean` first or make
might not realize anything needs to be rebuilt.

`make` - will package up the game into `bin/jasonsgame-[your-platform]-[public|internal]` and the frontend into `out/JasonsGame-[your-platform]-x64`.
    - You can run it with `-localnet=true` to connect to a local network; default is to connect to Quorum Control test net.
    - When you run the Electron frontend you can also connect to the backend via a web browser at `http://localhost:8080/`.
    
`make PLATFORM=all` - will build binaries and distributables for macOS, Linux, and Windows.
    - These "distributables" have an Electron frontend built in.

## Developing

These are written as short hand notes and will get fleshed out.

* Many things are in the Make file - explore there first :)
* A complete local env: `make localnet` `make game-server`
  * the first time you do this you'll also need to run a `make frontend-build`
  * Now you can see http://localhost:8080/ in your browser
* There are some "hacks" to make the docker dev run faster, you can see them in the docker-compose-dev.yml where we use the go pkg cache from your local box, and also cache go builds to your local box (rather than having to recreate all that every time in the container). This makes `make integration-test` not take a decade to startup.
* frontend uses re-frame and shadow-cljs with all the fixins (hot reload, reframe-10x, etc)
* the IPFS libp2p node has autorelay turned on (as it is in IPFS 0.4.20+ now), it also tries to discover other open games and directly connect to them. (network/discovery.go)
* ipldstore.go is a new TreeStore which is a combo nodestore and also an easy way to get/store the tree. This store is backed directly by an ipfslite instance, so nodes are stored locally *and* broadcast to the network as they are created. Additionally, there is a publish of the nodes (currently to infura).
* ipfslite package is mostly taken from here: https://github.com/hsanjuan/ipfs-lite but we need to be able to keep versions consistent across our libp2p interactions, and our usecase is actually simpler, so was able to strip away more from even the "lite" nodes.

### Hacking on the frontend

* `make frontend-dev` will launch a hot reloadable frontend - available at http://localhost:8280 (and pointing to the game server at localhost:8080)

### Two games
* `make game2` will launch a second game available at http://localhost:8090

### Against testnet

* `make game-server testnet=1` and then hit http://localhost:8080

### Ink

In production ink is scarce and supplied to the game manually and deliberately.
But in dev it's convenient for the (fake) ink to flow like the salmon of Capistrano!

There are a few moving parts to that and even more steps that need to happen in
the right order. `scripts/start-dev.sh` shows the details, but you probably
don't want to run that directly. 

Running `make dev` will spin everything up for you in a Docker environment.

You can run `make down` to ensure that it all gets torn down properly
(which CTRL-c doesn't always do).
