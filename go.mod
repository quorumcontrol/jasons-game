module github.com/quorumcontrol/jasons-game

go 1.12

require (
	github.com/AsynkronIT/protoactor-go v0.0.0-20190318154652-aa1aa20c2fa0
	github.com/btcsuite/btcd v0.0.0-20190427004231-96897255fd17 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc
	github.com/ethereum/go-ethereum v1.8.27
	github.com/gdamore/tcell v1.1.1
	github.com/golang/protobuf v1.3.1
	github.com/hsanjuan/ipfs-lite v0.0.3
	github.com/ipfs/go-bitswap v0.0.4
	github.com/ipfs/go-block-format v0.0.2
	github.com/ipfs/go-blockservice v0.0.3
	github.com/ipfs/go-cid v0.0.1
	github.com/ipfs/go-datastore v0.0.4
	github.com/ipfs/go-ds-badger v0.0.3
	github.com/ipfs/go-ipfs-blockstore v0.0.1
	github.com/ipfs/go-ipfs-config v0.0.2
	github.com/ipfs/go-ipfs-exchange-offline v0.0.1
	github.com/ipfs/go-ipfs-http-client v0.0.1 // indirect
	github.com/ipfs/go-ipld-cbor v1.5.1-0.20190302174746-59d816225550
	github.com/ipfs/go-ipld-format v0.0.1
	github.com/ipfs/go-log v0.0.1
	github.com/ipfs/go-merkledag v0.0.3
	github.com/jbenet/goprocess v0.0.0-20160826012719-b497e2f366b8
	github.com/libp2p/go-libp2p v0.0.21
	github.com/libp2p/go-libp2p-crypto v0.0.1
	github.com/libp2p/go-libp2p-discovery v0.0.2
	github.com/libp2p/go-libp2p-host v0.0.2
	github.com/libp2p/go-libp2p-kad-dht v0.0.10
	github.com/libp2p/go-libp2p-loggables v0.0.1
	github.com/libp2p/go-libp2p-net v0.0.2
	github.com/libp2p/go-libp2p-peer v0.1.0
	github.com/libp2p/go-libp2p-peerstore v0.0.5
	github.com/libp2p/go-libp2p-routing v0.0.1
	github.com/lucasb-eyer/go-colorful v1.0.2 // indirect
	github.com/multiformats/go-multiaddr v0.0.2
	github.com/multiformats/go-multihash v0.0.5 // indirect
	github.com/pkg/errors v0.8.1
	github.com/quorumcontrol/chaintree v0.0.0-20190426130059-dda329e6bd87
	github.com/quorumcontrol/storage v1.1.2
	github.com/quorumcontrol/tupelo-go-sdk v0.2.1-0.20190501192947-deae39695b92
	github.com/rivo/tview v0.0.0-20190406182340-90b4da1bd64c
	github.com/rivo/uniseg v0.0.0-20190313204849-f699dde9c340 // indirect
	github.com/sbstjn/allot v0.0.0-20161025071122-1f2349a
	github.com/stretchr/testify v1.3.0
	github.com/whyrusleeping/go-logging v0.0.0-20170515211332-0457bb6b88fc
	golang.org/x/net v0.0.0-20190415214537-1da14a5a36f2
	golang.org/x/sync v0.0.0-20190423024810-112230192c58 // indirect
	google.golang.org/grpc v1.20.0
)

replace github.com/quorumcontrol/tupelo-go-sdk v0.2.1-0.20190501192947-deae39695b92 => ../tupelo-go-client
