module github.com/quorumcontrol/jasons-game

go 1.12

require (
	github.com/AsynkronIT/protoactor-go v0.0.0-20190429152931-21e2d03dcae5
	github.com/aws/aws-sdk-go v1.15.60
	github.com/ethereum/go-ethereum v1.8.27
	github.com/gobuffalo/genny v0.1.1 // indirect
	github.com/gobuffalo/gogen v0.1.1 // indirect
	github.com/gobuffalo/packr/v2 v2.2.0
	github.com/gogo/protobuf v1.2.1
	github.com/golang/protobuf v1.3.1
	github.com/gorilla/mux v1.7.1
	github.com/improbable-eng/grpc-web v0.9.5
	github.com/ipfs/go-block-format v0.0.2
	github.com/ipfs/go-blockservice v0.1.0
	github.com/ipfs/go-cid v0.0.2
	github.com/ipfs/go-datastore v0.0.5
	github.com/ipfs/go-ds-badger v0.0.5
	github.com/ipfs/go-ds-s3 v0.0.1
	github.com/ipfs/go-ipfs-blockstore v0.0.1
	github.com/ipfs/go-ipfs-config v0.0.6
	github.com/ipfs/go-ipfs-ds-help v0.0.1
	github.com/ipfs/go-ipfs-exchange-offline v0.0.1
	github.com/ipfs/go-ipfs-http-client v0.0.3 // indirect
	github.com/ipfs/go-ipld-cbor v1.5.1-0.20190302174746-59d816225550
	github.com/ipfs/go-ipld-format v0.0.2
	github.com/ipfs/go-log v0.0.1
	github.com/ipfs/go-merkledag v0.1.0
	github.com/karrick/godirwalk v1.10.3 // indirect
	github.com/libp2p/go-libp2p v0.2.0
	github.com/libp2p/go-libp2p-connmgr v0.1.0
	github.com/libp2p/go-libp2p-core v0.0.6
	github.com/libp2p/go-libp2p-pubsub v0.1.0
	github.com/pkg/errors v0.8.1
	github.com/quorumcontrol/chaintree v0.0.0-20190701175144-f8f44c3e6d4b
	github.com/quorumcontrol/community v0.0.1
	github.com/quorumcontrol/messages/build/go v0.0.0-20190716095704-9acdbae78c93
	github.com/quorumcontrol/namedlocker v0.0.0-20180808140020-3f797c8b12b1 // indirect
	github.com/quorumcontrol/storage v1.1.4
	github.com/quorumcontrol/tupelo-go-sdk v0.5.1
	github.com/shibukawa/configdir v0.0.0-20170330084843-e180dbdc8da0
	github.com/sirupsen/logrus v1.4.2 // indirect
	github.com/spf13/cobra v0.0.5
	github.com/stretchr/testify v1.3.0
	github.com/zserge/webview v0.0.0-20190123072648-16c93bcaeaeb
	golang.org/x/crypto v0.0.0-20190618222545-ea8f1a30c443
	golang.org/x/net v0.0.0-20190620200207-3b0461eec859
	golang.org/x/tools v0.0.0-20190523174634-38d8bcfa38af // indirect
	google.golang.org/genproto v0.0.0-20190418145605-e7d98fc518a7 // indirect
	google.golang.org/grpc v1.20.0
)

replace github.com/libp2p/go-libp2p-pubsub v0.1.0 => github.com/quorumcontrol/go-libp2p-pubsub v0.0.4-0.20190528094025-e4e719f73e7a

// use our fork of packr until https://github.com/gobuffalo/packr/issues/198 is fixed
replace github.com/gobuffalo/packr/v2 v2.2.0 => github.com/quorumcontrol/packr/v2 v2.2.1-0.20190523180755-1b8140a3b2d5
