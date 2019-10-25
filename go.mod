module github.com/quorumcontrol/jasons-game

go 1.12

require (
	github.com/99designs/keyring v1.1.2
	github.com/AsynkronIT/protoactor-go v0.0.0-20190821183243-5bb73de32899
	github.com/aws/aws-lambda-go v1.13.2
	github.com/btcsuite/btcutil v0.0.0-20190425235716-9e5f4b9a998d
	github.com/dgraph-io/badger v1.6.0 // indirect
	github.com/ethereum/go-ethereum v1.9.3
	github.com/gobuffalo/packr/v2 v2.5.3-0.20190708182234-662c20c19dde
	github.com/gogo/protobuf v1.3.1
	github.com/google/uuid v1.1.1
	github.com/gorilla/mux v1.7.1
	github.com/hashicorp/go-uuid v1.0.1
	github.com/hashicorp/golang-lru v0.5.3
	github.com/imdario/mergo v0.3.7
	github.com/improbable-eng/grpc-web v0.11.0
	github.com/ipfs/go-blockservice v0.1.1
	github.com/ipfs/go-cid v0.0.3
	github.com/ipfs/go-datastore v0.1.1
	github.com/ipfs/go-ds-badger v0.0.5
	github.com/ipfs/go-ipfs-blockstore v0.1.0
	github.com/ipfs/go-ipfs-exchange-offline v0.0.1
	github.com/ipfs/go-ipld-cbor v0.0.3
	github.com/ipfs/go-ipld-format v0.0.2
	github.com/ipfs/go-log v0.0.1
	github.com/ipfs/go-merkledag v0.1.0
	github.com/libp2p/go-libp2p-core v0.2.3
	github.com/montanaflynn/stats v0.5.0
	github.com/mr-tron/base58 v1.1.2
	github.com/multiformats/go-multiaddr v0.1.1
	github.com/mwitkow/go-conntrack v0.0.0-20161129095857-cc309e4a2223 // indirect
	github.com/pkg/errors v0.8.1
	github.com/quorumcontrol/chaintree v0.8.6-0.20191007111216-51a819c15c38
	github.com/quorumcontrol/community v0.0.3-0.20190924213249-7b989784c22d
	github.com/quorumcontrol/messages/build/go v0.0.0-20190916172743-fed64641cd55
	github.com/quorumcontrol/tupelo-go-sdk v0.5.10-0.20191025142624-935d3e7cd723
	github.com/rs/cors v1.7.0 // indirect
	github.com/shibukawa/configdir v0.0.0-20170330084843-e180dbdc8da0
	github.com/spf13/cobra v0.0.5
	github.com/stretchr/testify v1.4.0
	github.com/tyler-smith/go-bip39 v1.0.2
	golang.org/x/crypto v0.0.0-20191011191535-87dc89f01550
	golang.org/x/net v0.0.0-20190827160401-ba9fcec4b297 // indirect
	google.golang.org/appengine v1.4.0 // indirect
	google.golang.org/grpc v1.22.0
	gopkg.in/yaml.v2 v2.2.2
)

replace github.com/libp2p/go-libp2p-pubsub v0.1.0 => github.com/quorumcontrol/go-libp2p-pubsub v0.0.4-0.20190528094025-e4e719f73e7a

replace github.com/gobuffalo/packr/v2 v2.5.1 => github.com/gobuffalo/packr/v2 v2.5.3-0.20190708182234-662c20c19dde

replace github.com/libp2p/go-libp2p-core => github.com/quorumcontrol/go-libp2p-core v0.2.4-0.20191017172042-69fe90d32d39
