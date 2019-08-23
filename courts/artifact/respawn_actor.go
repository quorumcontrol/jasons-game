package artifact

import (
	"context"
	"encoding/binary"
	"fmt"
	"math/rand"
	"path/filepath"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/jasons-game/courts/config"
	"github.com/quorumcontrol/jasons-game/game/trees"
	"github.com/quorumcontrol/jasons-game/importer"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/messages/build/go/signatures"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
)

var log = logging.Logger("respawner")

type RespawnActor struct {
	parentCtx        context.Context
	pid              *actor.PID
	currentObjectTip string
	stateSubscriber  *actor.PID
	network          network.Network
	locations        []string
	forgers          []string
	cfg              *artifactsConfig
}

type RespawnActorConfig struct {
	Network    network.Network
	Locations  []string
	Forgers    []string
	ConfigPath string
}

func NewRespawnActor(ctx context.Context, cfg *RespawnActorConfig) (*RespawnActor, error) {
	artifactCfg, err := newArtifactsConfig(filepath.Join(cfg.ConfigPath, "artifacts"))
	if err != nil {
		return nil, err
	}

	return &RespawnActor{
		parentCtx: ctx,
		network:   cfg.Network,
		locations: cfg.Locations,
		forgers:   cfg.Forgers,
		cfg:       artifactCfg,
	}, nil
}

func (r *RespawnActor) Start(actorCtx actor.Context) {
	r.pid = actorCtx.Spawn(actor.PropsFromProducer(func() actor.Actor {
		return r
	}))

	go func() {
		<-r.parentCtx.Done()
		actorCtx.Stop(r.pid)
	}()
}

func (r *RespawnActor) PID() *actor.PID {
	return r.pid
}

func (r *RespawnActor) respawnTree(actorCtx actor.Context) (*consensus.SignedChainTree, error) {
	treeKey, err := consensus.PassPhraseKey(crypto.FromECDSA(r.network.PrivateKey()), []byte("artifact-respawner"))
	if err != nil {
		return nil, errors.Wrap(err, "error creating new respwaner key")
	}
	treeDid := consensus.EcdsaPubkeyToDid(treeKey.PublicKey)

	log.Debugf("respawnTree looking for %s", treeDid)
	tree, err := r.network.GetTree(treeDid)
	if err != nil {
		return nil, errors.Wrap(err, "error fetching latest object tip")
	}

	if tree == nil {
		log.Debugf("respawnTree %s does not exist yet, creating", treeDid)

		tree, err = r.network.CreateChainTreeWithKey(treeKey)
		if err != nil {
			return nil, errors.Wrap(err, "error creating new respwaner tree")
		}

		tree, err = r.network.ChangeChainTreeOwnerWithKey(tree, treeKey, []string{
			crypto.PubkeyToAddress(*r.network.PublicKey()).String(),
		})
		if err != nil {
			return nil, errors.Wrap(err, "error chowning new respwaner")
		}

		log.Debugf("respawnTree %s created", treeDid)
	}

	return tree, nil
}

func (r *RespawnActor) generateRandomArtifact(salt []byte) (*consensus.SignedChainTree, error) {
	treeKey, err := consensus.PassPhraseKey(crypto.FromECDSA(r.network.PrivateKey()), salt)
	if err != nil {
		return nil, errors.Wrap(err, "error creating new object key")
	}

	did := consensus.EcdsaPubkeyToDid(treeKey.PublicKey)

	// Pick object to spawn deterministically based on chaintree id
	randSeed := binary.BigEndian.Uint64([]byte(consensus.DidToAddr(did)[2:])[0:8])
	log.Debugf("spawnObject rand seed %d", randSeed)
	random := rand.New(rand.NewSource(int64(randSeed)))

	randomObjectName := "artifact" + r.cfg.NamesPool[random.Intn(len(r.cfg.NamesPool))]

	variables := map[string]interface{}{
		"Name": randomObjectName,
		"Did":  did,
	}

	inscribeableKeys := r.cfg.inscribeableKeys()
	for _, inscriptionKey := range inscribeableKeys {
		// Andrew wanted blanks in here 2/7 of time
		inscribeableValues := append(r.cfg.inscribeableValuesFor(inscriptionKey), []string{"", ""}...)
		randomInscriptionValue := inscribeableValues[random.Intn(len(inscribeableValues))]
		variables[inscriptionKey] = randomInscriptionValue
	}
	variables["ForgedBy"] = r.forgers[random.Intn(len(r.forgers))]

	processedYaml, err := config.ReplaceVariables(string(r.cfg.ObjectTemplate), variables)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing ObjectTemplate")
	}

	tree, err := r.network.CreateChainTreeWithKey(treeKey)
	if err != nil {
		return nil, errors.Wrap(err, "error creating new object tree")
	}

	_, err = r.network.ChangeChainTreeOwnerWithKey(tree, treeKey, []string{
		crypto.PubkeyToAddress(*r.network.PublicKey()).String(),
	})
	if err != nil {
		return nil, errors.Wrap(err, "error chowning new object")
	}

	err = importer.New(r.network).UpdateObject(did, processedYaml)
	if err != nil {
		return nil, err
	}

	// Fetch latest chaintree
	tree, err = r.network.GetTree(did)
	if err != nil {
		return nil, errors.Wrap(err, "error fetching latest object tip")
	}

	return tree, nil
}

func (r *RespawnActor) addArtifactToRandomLocation(artifactDid string) (string, error) {
	randSeed := binary.BigEndian.Uint64([]byte(consensus.DidToAddr(artifactDid)[2:])[0:8])
	random := rand.New(rand.NewSource(int64(randSeed)))
	locToSpawnIn := r.locations[random.Intn(len(r.locations))]

	locTree, err := r.network.GetTree(locToSpawnIn)
	if err != nil {
		return "", errors.Wrap(err, fmt.Sprintf("error getting spawn location %s", locToSpawnIn))
	}
	locInventory := trees.NewInventoryTree(r.network, locTree)
	err = locInventory.Add(artifactDid)
	if err != nil {
		return "", errors.Wrap(err, fmt.Sprintf("error spawning object at location %s", locToSpawnIn))
	}

	return locTree.MustId(), nil
}

func (r *RespawnActor) spawnObject(actorCtx actor.Context) error {
	respawnTree, err := r.respawnTree(actorCtx)
	if err != nil {
		return errors.Wrap(err, "error fetching respawnTree")
	}

	// use previous respawn tree tip to allow this service to be deterministic and stateless
	artifact, err := r.generateRandomArtifact(respawnTree.Tip().Bytes())
	if err != nil {
		return errors.Wrap(err, "error generating artifact")
	}

	r.currentObjectTip = artifact.Tip().String()

	if r.stateSubscriber != nil {
		actorCtx.Stop(r.stateSubscriber)
	}

	r.stateSubscriber = actorCtx.Spawn(r.network.NewCurrentStateSubscriptionProps(artifact.MustId()))

	spawnedLocationDid, err := r.addArtifactToRandomLocation(artifact.MustId())
	if err != nil {
		return errors.Wrap(err, "error adding artifact to location")
	}

	// TODO: does this need to be encrypted?
	_, err = r.network.UpdateChainTree(respawnTree, "last", map[string]string{
		"did":      artifact.MustId(),
		"location": spawnedLocationDid,
	})
	if err != nil {
		return errors.Wrap(err, "error updating respawnTree")
	}

	log.Infof("spawnObject new object spawned %s into %s", artifact.MustId(), spawnedLocationDid)
	return nil
}

func (r *RespawnActor) Receive(actorCtx actor.Context) {
	switch msg := actorCtx.Message().(type) {
	case *actor.Started:
		log.Debugf("init: starting")

		respawnTree, err := r.respawnTree(actorCtx)
		if err != nil {
			panic(errors.Wrap(err, "error fetching respawnTree"))
		}

		log.Debugf("init: cfgured")

		lastUncast, _, err := respawnTree.ChainTree.Dag.Resolve(r.parentCtx, []string{"tree", "data", "last"})
		if err != nil {
			panic(errors.Wrap(err, "error fetching respawnTree data"))
		}
		if lastUncast != nil {
			last, _ := lastUncast.(map[string]interface{})
			locationDid := last["location"].(string)
			objectDid := last["did"].(string)

			locTree, err := r.network.GetTree(locationDid)
			if err != nil {
				panic(errors.Wrap(err, "error getting last spawn location"))
			}
			locInventory := trees.NewInventoryTree(r.network, locTree)

			currentObj, err := r.network.GetTree(objectDid)
			if err != nil {
				panic(errors.Wrap(err, "error fetching latest object tip"))
			}

			r.currentObjectTip = currentObj.Tip().String()

			exists, err := locInventory.Exists(currentObj.MustId())
			if err != nil {
				panic(errors.Wrap(err, "error checking exists"))
			}

			// Previously spawned object is still valid, attach subscriber and finish init
			if exists {
				log.Infof("object %s already exists at %s\n", currentObj.MustId(), locationDid)
				r.stateSubscriber = actorCtx.Spawn(r.network.NewCurrentStateSubscriptionProps(currentObj.MustId()))
				return
			}
		}

		err = r.spawnObject(actorCtx)
		if err != nil {
			panic(err)
		}
	case *signatures.CurrentState:
		previousCid, _ := cid.Parse(msg.Signature.PreviousTip)
		// object has changed, spawn new object
		if previousCid.String() == r.currentObjectTip {
			err := r.spawnObject(actorCtx)
			if err != nil {
				panic(err)
			}
		}
	}
}
