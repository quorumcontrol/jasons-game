module github.com/quorumcontrol/jasons-game

go 1.12

require (
	github.com/AsynkronIT/protoactor-go v0.0.0-20190821183243-5bb73de32899
	github.com/btcsuite/btcutil v0.0.0-20190425235716-9e5f4b9a998d
	github.com/dgraph-io/badger v1.6.0 // indirect
	github.com/ethereum/go-ethereum v1.8.27
	github.com/go-stack/stack v1.8.0 // indirect
	github.com/gobuffalo/packr/v2 v2.5.3-0.20190708182234-662c20c19dde
	github.com/gogo/protobuf v1.2.1
	github.com/google/uuid v1.1.1
	github.com/gorilla/mux v1.7.1
	github.com/hashicorp/go-uuid v1.0.1
	github.com/imdario/mergo v0.3.7
	github.com/improbable-eng/grpc-web v0.9.5
	github.com/ipfs/go-blockservice v0.1.1
	github.com/ipfs/go-cid v0.0.2
	github.com/ipfs/go-datastore v0.0.5
	github.com/ipfs/go-ds-badger v0.0.5
	github.com/ipfs/go-ipfs-blockstore v0.0.1
	github.com/ipfs/go-ipfs-config v0.0.6
	github.com/ipfs/go-ipfs-exchange-offline v0.0.1
	github.com/ipfs/go-ipld-cbor v1.5.1-0.20190302174746-59d816225550
	github.com/ipfs/go-ipld-format v0.0.2
	github.com/ipfs/go-log v0.0.1
	github.com/ipfs/go-merkledag v0.1.0
	github.com/libp2p/go-libp2p-transport v0.0.5 // indirect
	github.com/libp2p/go-testutil v0.1.0 // indirect
	github.com/mr-tron/base58 v1.1.2
	github.com/pkg/errors v0.8.1
	github.com/quorumcontrol/chaintree v0.0.0-20190709145156-03b818830f38
	github.com/quorumcontrol/community v0.0.0-20190722000000-5f668465bdd9673c3cfbe29771e54f221432cf9c
	github.com/quorumcontrol/messages/build/go v0.0.0-20190723140659-c8a3565a7c72
	github.com/quorumcontrol/tupelo-go-sdk v0.5.3-rc2.0.20190729193835-fd24148756d5
	github.com/rs/cors v1.7.0 // indirect
	github.com/shibukawa/configdir v0.0.0-20170330084843-e180dbdc8da0
	github.com/spf13/cobra v0.0.5
	github.com/stretchr/testify v1.3.0
	golang.org/x/crypto v0.0.0-20190701094942-4def268fd1a4
	google.golang.org/grpc v1.22.0
	gopkg.in/yaml.v2 v2.2.2
	gotest.tools/gotestsum v0.3.5 // indirect
)

replace github.com/libp2p/go-libp2p-pubsub v0.1.0 => github.com/quorumcontrol/go-libp2p-pubsub v0.0.4-0.20190528094025-e4e719f73e7a

replace github.com/gobuffalo/packr/v2 v2.5.1 => github.com/gobuffalo/packr/v2 v2.5.3-0.20190708182234-662c20c19dde
