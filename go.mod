module github.com/quorumcontrol/jasons-game

go 1.12

require (
	github.com/99designs/keyring v1.1.2
	github.com/AsynkronIT/protoactor-go v0.0.0-20190821183243-5bb73de32899
	github.com/FactomProject/basen v0.0.0-20150613233007-fe3947df716e // indirect
	github.com/FactomProject/btcutilecc v0.0.0-20130527213604-d3a63a5752ec // indirect
	github.com/apex/log v1.1.1
	github.com/aws/aws-lambda-go v1.13.2
	github.com/btcsuite/btcutil v0.0.0-20190425235716-9e5f4b9a998d
	github.com/dgraph-io/badger v1.6.0 // indirect
	github.com/ethereum/go-ethereum v1.9.3
	github.com/gobuffalo/packr/v2 v2.5.3-0.20190708182234-662c20c19dde
	github.com/gogo/protobuf v1.3.0
	github.com/google/uuid v1.1.1
	github.com/gorilla/mux v1.7.1
	github.com/hashicorp/go-uuid v1.0.1
	github.com/hashicorp/golang-lru v0.5.1
	github.com/imdario/mergo v0.3.7
	github.com/improbable-eng/grpc-web v0.11.0
	github.com/ipfs/go-blockservice v0.1.1
	github.com/ipfs/go-cid v0.0.2
	github.com/ipfs/go-datastore v0.0.5
	github.com/ipfs/go-ds-badger v0.0.5
	github.com/ipfs/go-ipfs-blockstore v0.0.1
	github.com/ipfs/go-ipfs-config v0.0.6 // indirect
	github.com/ipfs/go-ipfs-exchange-offline v0.0.1
	github.com/ipfs/go-ipld-cbor v0.0.3
	github.com/ipfs/go-ipld-format v0.0.2
	github.com/ipfs/go-log v0.0.1
	github.com/ipfs/go-merkledag v0.1.0
	github.com/montanaflynn/stats v0.5.0
	github.com/mr-tron/base58 v1.1.2
	github.com/mwitkow/go-conntrack v0.0.0-20161129095857-cc309e4a2223 // indirect
	github.com/pkg/errors v0.8.1
	github.com/quorumcontrol/chaintree v0.8.4
	github.com/quorumcontrol/community v0.0.3-0.20190924213249-7b989784c22d
	github.com/quorumcontrol/messages/build/go v0.0.0-20190916172743-fed64641cd55
	github.com/quorumcontrol/tupelo-go-sdk v0.5.6-0.20190924212204-7a51b39b27bb
	github.com/rs/cors v1.7.0 // indirect
	github.com/shibukawa/configdir v0.0.0-20170330084843-e180dbdc8da0
	github.com/spf13/cobra v0.0.5
	github.com/stretchr/testify v1.4.0
	github.com/tyler-smith/go-bip32 v0.0.0-20170922074101-2c9cfd177564
	github.com/tyler-smith/go-bip39 v1.0.2
	golang.org/x/crypto v0.0.0-20190829043050-9756ffdc2472
	golang.org/x/net v0.0.0-20190827160401-ba9fcec4b297 // indirect
	golang.org/x/sys v0.0.0-20190902133755-9109b7679e13 // indirect
	google.golang.org/appengine v1.4.0 // indirect
	google.golang.org/grpc v1.22.0
	gopkg.in/yaml.v2 v2.2.2
)

replace github.com/libp2p/go-libp2p-pubsub v0.1.0 => github.com/quorumcontrol/go-libp2p-pubsub v0.0.4-0.20190528094025-e4e719f73e7a

replace github.com/gobuffalo/packr/v2 v2.5.1 => github.com/gobuffalo/packr/v2 v2.5.3-0.20190708182234-662c20c19dde
