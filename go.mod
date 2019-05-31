module github.com/quorumcontrol/jasons-game

go 1.12

require (
	github.com/AsynkronIT/protoactor-go v0.0.0-20190429152931-21e2d03dcae5
	github.com/aws/aws-sdk-go v1.15.60
	github.com/btcsuite/btcd v0.0.0-20190427004231-96897255fd17 // indirect
	github.com/ethereum/go-ethereum v1.8.27
	github.com/gobuffalo/genny v0.1.1 // indirect
	github.com/gobuffalo/gogen v0.1.1 // indirect
	github.com/gobuffalo/packr/v2 v2.2.0
	github.com/gogo/protobuf v1.2.1
	github.com/gorilla/mux v1.7.1
	github.com/improbable-eng/grpc-web v0.9.5
	github.com/ipfs/go-block-format v0.0.2
	github.com/ipfs/go-blockservice v0.0.3
	github.com/ipfs/go-cid v0.0.1
	github.com/ipfs/go-datastore v0.0.5
	github.com/ipfs/go-ds-badger v0.0.3
	github.com/ipfs/go-ds-s3 v0.0.1
	github.com/ipfs/go-ipfs-blockstore v0.0.1
	github.com/ipfs/go-ipfs-config v0.0.2
	github.com/ipfs/go-ipfs-exchange-offline v0.0.1
	github.com/ipfs/go-ipfs-http-client v0.0.1 // indirect
	github.com/ipfs/go-ipld-cbor v1.5.1-0.20190302174746-59d816225550
	github.com/ipfs/go-ipld-format v0.0.1
	github.com/ipfs/go-log v0.0.1
	github.com/ipfs/go-merkledag v0.0.3
	github.com/karrick/godirwalk v1.10.3 // indirect
	github.com/libp2p/go-libp2p v0.0.21
	github.com/libp2p/go-libp2p-connmgr v0.0.3
	github.com/libp2p/go-libp2p-interface-connmgr v0.0.3
	github.com/libp2p/go-libp2p-peer v0.1.0
	github.com/libp2p/go-libp2p-pubsub v0.0.3
	github.com/multiformats/go-multihash v0.0.5 // indirect
	github.com/pkg/errors v0.8.1
	github.com/quorumcontrol/chaintree v0.0.0-20190515172607-6a3407e067bd
	github.com/quorumcontrol/storage v1.1.2
	github.com/quorumcontrol/tupelo-go-sdk v0.2.4-0.20190517142847-a8abc862d8a0dd497e2630b0657dec428d86f1a2
	github.com/shibukawa/configdir v0.0.0-20170330084843-e180dbdc8da0
	github.com/sirupsen/logrus v1.4.2 // indirect
	github.com/stretchr/testify v1.3.0
	github.com/tinylib/msgp v1.1.0
	github.com/zserge/webview v0.0.0-20190123072648-16c93bcaeaeb
	golang.org/x/crypto v0.0.0-20190530122614-20be4c3c3ed5 // indirect
	golang.org/x/net v0.0.0-20190522155817-f3200d17e092 // indirect
	golang.org/x/sys v0.0.0-20190531175056-4c3a928424d2 // indirect
	golang.org/x/text v0.3.2 // indirect
	golang.org/x/tools v0.0.0-20190531172133-b3315ee88b7d // indirect
	google.golang.org/grpc v1.20.0
)

replace github.com/libp2p/go-libp2p-pubsub v0.0.3 => github.com/quorumcontrol/go-libp2p-pubsub v0.0.0-20190515123400-58d894b144ff864d212cf4b13c42e8fdfe783aba

// use our fork of packr until https://github.com/gobuffalo/packr/issues/198 is fixed
replace github.com/gobuffalo/packr/v2 v2.2.0 => github.com/quorumcontrol/packr/v2 v2.2.1-0.20190523180755-1b8140a3b2d5
