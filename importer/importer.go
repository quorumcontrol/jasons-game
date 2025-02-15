package importer

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"unicode"

	"github.com/ethereum/go-ethereum/common/hexutil"
	ptypes "github.com/gogo/protobuf/types"
	"github.com/imdario/mergo"
	logging "github.com/ipfs/go-log"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/chaintree/typecaster"
	"github.com/quorumcontrol/jasons-game/game"
	"github.com/quorumcontrol/jasons-game/game/static"
	"github.com/quorumcontrol/jasons-game/game/trees"
	"github.com/quorumcontrol/jasons-game/importer/flatmap"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"gopkg.in/yaml.v2"
)

var log = logging.Logger("importer")

type nameToDidMap map[string]string

type NameToDids struct {
	Locations nameToDidMap
	Objects   nameToDidMap
	Static    nameToDidMap
}

type ImportInteraction struct {
	Type  string                 `yaml:"type"`
	Value map[string]interface{} `yaml:"value"`
}

type ImportLocation struct {
	Data         map[string]interface{} `yaml:"data"`
	Interactions []*ImportInteraction   `yaml:"interactions"`
	Inventory    []string               `yaml:"inventory"`
}

type ImportObject struct {
	Data         map[string]interface{} `yaml:"data"`
	Interactions []*ImportInteraction   `yaml:"interactions"`
}

type ImportPayload struct {
	Locations map[string]*ImportLocation `yaml:"locations"`
	Objects   map[string]*ImportObject   `yaml:"objects"`
}

type Importer struct {
	network network.Network
}

func New(network network.Network) *Importer {
	return &Importer{
		network: network,
	}
}

func (i *Importer) createTrees(data *ImportPayload) (*NameToDids, error) {
	staticVals, err := static.GetAll(i.network)
	if err != nil {
		return nil, err
	}

	ids := &NameToDids{
		Locations: make(nameToDidMap),
		Objects:   make(nameToDidMap),
		Static:    staticVals,
	}

	for key := range data.Locations {
		tree, err := i.network.FindOrCreatePassphraseTree("locations/" + key)
		if err != nil {
			return nil, err
		}
		ids.Locations[key] = tree.MustId()
		log.Infof("%s: Created chaintree for locations.%s", tree.MustId(), key)
	}

	for key := range data.Objects {
		tree, err := i.network.FindOrCreatePassphraseTree("objects/" + key)
		if err != nil {
			return nil, err
		}
		ids.Objects[key] = tree.MustId()
		log.Infof("%s: Created chaintree for objects.%s", tree.MustId(), key)
	}

	return ids, nil
}

func (i *Importer) loadBasicData(tree *consensus.SignedChainTree, data map[string]interface{}) (*consensus.SignedChainTree, error) {
	var err error
	if len(data) == 0 {
		return tree, err
	}

	for _, reservedKey := range reservedKeys {
		if _, ok := data[reservedKey]; ok {
			return tree, fmt.Errorf("error reserved key: can't set %v in data as top level attr", reservedKey)
		}
	}

	flatPaths := flatmap.Flatten(data)

	// important! order of UpdateChainTree matters, so make sure there is
	// consistent ordering
	// beware of the age old gotcha, go map iteration order isn't guaranteed
	sortedKeys := flatPaths.Keys()
	sort.Strings(sortedKeys)

	for _, key := range sortedKeys {
		val := flatPaths[key]
		tree, err = i.network.UpdateChainTree(tree, fmt.Sprintf("jasons-game/%s", key), val)

		if err != nil {
			return tree, errors.Wrap(err, "updating location data")
		}
		log.Debugf("%v: data: set path %s to %v", tree.MustId(), key, val)
	}

	return tree, nil
}

func (i *Importer) loadInventory(tree *consensus.SignedChainTree, data []string) (*consensus.SignedChainTree, error) {
	var err error
	if len(data) == 0 {
		return tree, err
	}

	inventoryTree := trees.NewInventoryTree(i.network, tree)

	for _, objectDid := range data {
		err = inventoryTree.Add(objectDid)
		if err != nil {
			return inventoryTree.Tree(), err
		}

		log.Debugf("%v: inventory: added %s", tree.MustId(), objectDid)
	}

	return inventoryTree.Tree(), nil
}

func (i *Importer) yamlTypecast(data interface{}, t interface{}) error {
	var asYaml []byte
	var err error

	switch dataCast := data.(type) {
	case []byte:
		asYaml = dataCast
	case string:
		asYaml = []byte(dataCast)
	default:
		asYaml, err = yaml.Marshal(data)
		if err != nil {
			return err
		}
	}

	err = yaml.Unmarshal(asYaml, t)
	if err != nil {
		return err
	}
	return nil
}

func (i *Importer) convertImportInteraction(attrs *ImportInteraction) (game.Interaction, error) {
	var interaction game.Interaction

	if attrs.Type == "" {
		return interaction, fmt.Errorf("Interaction %v must have type set", attrs)
	}

	switch attrs.Type {
	case "CipherInteraction":
		command, ok := attrs.Value["command"].(string)
		if !ok {
			return interaction, fmt.Errorf("CipherInteraction must have command")
		}
		secret, ok := attrs.Value["secret"].(string)
		if !ok {
			return interaction, fmt.Errorf("CipherInteraction must have secret")
		}

		successImportInteractionUncast, ok := attrs.Value["success_interaction"]
		if !ok {
			return interaction, fmt.Errorf("CipherInteraction must have success_interaction")
		}

		var successImportInteraction *ImportInteraction
		err := i.yamlTypecast(successImportInteractionUncast, &successImportInteraction)
		if err != nil {
			return interaction, fmt.Errorf("CipherInteraction success_interaction must be ImportInteraction")
		}

		successInteraction, err := i.convertImportInteraction(successImportInteraction)
		if err != nil {
			return interaction, err
		}

		failureImportInteractionUncast, ok := attrs.Value["failure_interaction"]
		if !ok {
			return interaction, fmt.Errorf("CipherInteraction must have failure_interaction")
		}

		var failureImportInteraction *ImportInteraction
		err = i.yamlTypecast(failureImportInteractionUncast, &failureImportInteraction)
		if err != nil {
			return interaction, fmt.Errorf("CipherInteraction failure_interaction must be ImportInteraction")
		}
		if _, ok := failureImportInteraction.Value["command"]; !ok {
			failureImportInteraction.Value["command"] = command
		}

		failureInteraction, err := i.convertImportInteraction(failureImportInteraction)
		if err != nil {
			return interaction, err
		}

		interaction, err = game.NewCipherInteraction(command, secret, successInteraction, failureInteraction)
		if err != nil {
			return interaction, errors.Wrap(err, "error creating CipherInteraction")
		}
	case "ChainedInteraction":
		command, ok := attrs.Value["command"].(string)
		if !ok {
			return interaction, fmt.Errorf("ChainedInteraction must have command")
		}
		interactionsUncast, ok := attrs.Value["interactions"]
		if !ok {
			return interaction, fmt.Errorf("ChainedInteraction must have an array of interactions")
		}

		interactionSliceUncast, ok := interactionsUncast.([]interface{})
		if !ok || len(interactionSliceUncast) == 0 {
			return interaction, fmt.Errorf("ChainedInteraction must have one or more valid Interactions")
		}

		interactions := make([]game.Interaction, len(interactionSliceUncast))
		for idx, interactionUncast := range interactionSliceUncast {
			var importInteraction *ImportInteraction
			err := i.yamlTypecast(interactionUncast, &importInteraction)
			if err != nil {
				return interaction, fmt.Errorf("ChainedInteraction interaction %d must be ImportInteraction", i)
			}
			if _, ok := importInteraction.Value["command"]; !ok {
				importInteraction.Value["command"] = command
			}
			interaction, err := i.convertImportInteraction(importInteraction)
			if err != nil {
				return interaction, err
			}
			interactions[idx] = interaction
		}

		var err error
		interaction, err = game.NewChainedInteraction(command, interactions...)
		if err != nil {
			return interaction, errors.Wrap(err, "error creating ChainedInteraction")
		}
	default:
		typeURL := fmt.Sprintf("type.googleapis.com/jasonsgame.%s", attrs.Type)

		anyInteraction, err := ptypes.EmptyAny(&ptypes.Any{TypeUrl: typeURL})
		if err != nil {
			return interaction, fmt.Errorf("protobuf type %v not found: %v", typeURL, err)
		}

		err = typecaster.ToType(attrs.Value, anyInteraction)
		if err != nil {
			return interaction, errors.Wrap(err, "error casting interaction")
		}

		var ok bool
		interaction, ok = anyInteraction.(game.Interaction)
		if !ok {
			return interaction, fmt.Errorf("%v is not an interaction", err)
		}
	}

	return interaction, nil
}

func (i *Importer) loadInteractions(tree *consensus.SignedChainTree, data []*ImportInteraction) (*consensus.SignedChainTree, error) {
	var err error
	if len(data) == 0 {
		return tree, err
	}

	interactionTree := game.NewInteractionTree(i.network, tree)

	for _, attrs := range data {
		interaction, err := i.convertImportInteraction(attrs)
		if err != nil {
			return interactionTree.Tree(), err
		}

		err = interactionTree.AddInteraction(interaction)
		if err != nil {
			return interactionTree.Tree(), err
		}

		log.Debugf("%v: interactions: added %s as '%s'", tree.MustId(), attrs.Type, interaction.GetCommand())
	}

	return interactionTree.Tree(), nil
}

var reservedKeys = []string{"inventory", "interactions"}

func (i *Importer) loadLocations(data map[string]*ImportLocation, ids *NameToDids) error {
	for name, locData := range data {
		did := ids.Locations[name]
		err := i.updateLocation(did, locData)
		if err != nil {
			return err
		}
	}
	return nil
}

func (i *Importer) updateLocation(did string, locData *ImportLocation) error {
	return i.updateTreeIfChanged(did, locData, func(tree *consensus.SignedChainTree) error {
		var err error

		tree, err = i.loadBasicData(tree, locData.Data)
		if err != nil {
			return err
		}

		tree, err = i.loadInteractions(tree, locData.Interactions)
		if err != nil {
			return err
		}

		_, err = i.loadInventory(tree, locData.Inventory)
		if err != nil {
			return err
		}

		return nil
	})
}

func (i *Importer) loadObjects(data map[string]*ImportObject, ids *NameToDids) error {
	for name, objData := range data {
		did := ids.Objects[name]

		if _, ok := objData.Data["name"]; !ok {
			// Files must be named with underscore, but default name in the UI should be hyphenated
			objData.Data["name"] = strings.ReplaceAll(name, "_", "-")
		}

		err := i.updateObject(did, objData)
		if err != nil {
			return err
		}
	}
	return nil
}

func (i *Importer) updateObject(did string, objData *ImportObject) error {
	return i.updateTreeIfChanged(did, objData, func(tree *consensus.SignedChainTree) error {
		var err error
		tree, err = i.loadBasicData(tree, objData.Data)
		if err != nil {
			return err
		}

		_, err = i.loadInteractions(tree, objData.Interactions)
		if err != nil {
			return err
		}
		return nil
	})
}

func (i *Importer) updateTreeIfChanged(did string, treeData interface{}, updateFunction func(tree *consensus.SignedChainTree) error) error {
	tree, err := i.network.GetTree(did)
	if err != nil {
		return err
	}

	importHashPath := []string{"tree", "data", "import-hash"}
	hashVal, _, err := tree.ChainTree.Dag.Resolve(context.Background(), importHashPath)
	if err != nil {
		return err
	}

	treeDataYaml, err := yaml.Marshal(treeData)
	if err != nil {
		return err
	}
	newHash := sha256.Sum256(treeDataYaml)
	newHashStr := hexutil.Encode(newHash[:32])

	hashValStr, hashValStrOk := hashVal.(string)

	// needs updated
	if !hashValStrOk || newHashStr != hashValStr {
		// reset tree back to an empty state for reimporting
		tree, err = i.network.UpdateChainTree(tree, "", make(map[string]interface{}))
		if err != nil {
			return err
		}

		// call update function for new data
		err = updateFunction(tree)
		if err != nil {
			return err
		}

		// get latest tree
		tree, err = i.network.GetTree(did)
		if err != nil {
			return err
		}

		// set new hash
		_, err = i.network.UpdateChainTree(tree, strings.Join(importHashPath[2:], "/"), newHashStr)
		if err != nil {
			return err
		}
	}

	return nil
}

func (i *Importer) replaceVariables(data *ImportPayload, vars interface{}) (*ImportPayload, error) {
	fullYaml, err := yaml.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmpl, err := template.New("import").Parse(string(fullYaml))
	if err != nil {
		return nil, err
	}

	var outBuff bytes.Buffer
	if err := tmpl.Execute(&outBuff, vars); err != nil {
		return nil, err
	}

	processedYaml := &ImportPayload{}
	err = yaml.Unmarshal(outBuff.Bytes(), processedYaml)
	if err != nil {
		return nil, err
	}

	return processedYaml, nil
}

func (i *Importer) UpdateObject(did string, objectData interface{}) error {
	importObject := &ImportObject{}
	err := i.yamlTypecast(objectData, importObject)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("error typecasting for %s", did))
	}
	err = i.updateObject(did, importObject)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("error updating %s", did))
	}
	return nil
}

func (i *Importer) UpdateLocation(did string, locationData interface{}) error {
	importLocation := &ImportLocation{}
	err := i.yamlTypecast(locationData, importLocation)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("error typecasting for %s", did))
	}
	err = i.updateLocation(did, importLocation)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("error updating %s", did))
	}
	return nil
}

func (i *Importer) Import(importPath string) (*NameToDids, error) {
	var err error
	var data *ImportPayload

	data, err = i.loadYaml(importPath)

	if err != nil {
		return nil, err
	}

	ids, err := i.createTrees(data)
	if err != nil {
		return ids, err
	}

	data, err = i.replaceVariables(data, ids)
	if err != nil {
		return ids, err
	}

	// Note: its important to load objects first since locations' inventory fetch
	// the name of an object by did
	err = i.loadObjects(data.Objects, ids)
	if err != nil {
		return ids, err
	}

	err = i.loadLocations(data.Locations, ids)
	if err != nil {
		return ids, err
	}

	log.Infof("import complete - %d locations created - %d objects created", len(ids.Locations), len(ids.Objects))

	return ids, nil
}

func (i *Importer) loadYaml(importPath string) (*ImportPayload, error) {
	loaded := make(map[string]interface{})

	err := filepath.Walk(importPath, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		// If importing a single file, treat it as the full compiled loaded item
		if importPath == p {
			yamlFile, err := ioutil.ReadFile(p)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("error reading file %s", p))
			}
			err = yaml.Unmarshal(yamlFile, loaded)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("error unmarshalling file %s", p))
			}
			return nil
		}

		built := make(map[string]interface{})
		working := built

		directoryPath, fileName := filepath.Split(p)
		fileNameWithoutExt := strings.TrimSuffix(fileName, filepath.Ext(fileName))

		if filepath.Ext(fileName) != ".yaml" && filepath.Ext(fileName) != ".yml" {
			log.Debugf("Skipping non-yaml file %s\n", p)
			return nil
		}

		if !isAlphaNumeric(fileNameWithoutExt) {
			log.Errorf("Filename %s must only contain alphanumeric or _ characters", p)
			panic("")
		}

		trimmedDirectoryPath := strings.Trim(strings.TrimPrefix(directoryPath, importPath), string(os.PathSeparator))
		directorySlice := []string{}
		if len(trimmedDirectoryPath) > 0 {
			directorySlice = strings.Split(trimmedDirectoryPath, string(os.PathSeparator))
		}

		for _, part := range directorySlice {
			if _, ok := working[part]; !ok {
				working[part] = make(map[string]interface{})
			}
			working = working[part].(map[string]interface{})
		}

		yamlData := make(map[interface{}]interface{})

		yamlFile, err := ioutil.ReadFile(p)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("error reading file %s", p))
		}
		err = yaml.Unmarshal(yamlFile, yamlData)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("error unmarshalling file %s", p))
		}

		working[fileNameWithoutExt] = yamlData

		if err := mergo.Merge(&loaded, built); err != nil {
			return errors.Wrap(err, fmt.Sprintf("error appending file %s", p))
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	processedYaml := &ImportPayload{}
	err = i.yamlTypecast(loaded, processedYaml)
	if err != nil {
		return nil, err
	}

	return processedYaml, nil
}

// Taken to ensure file name is alphanumeric, which is used in template parsing:
// https://github.com/golang/go/blob/c7bb4533cb7d91eadc9c674e48dc644bc831e64e/src/text/template/parse/lex.go#L664
func isAlphaNumeric(str string) bool {
	for _, r := range str {
		isValid := r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
		if !isValid {
			return false
		}
	}
	return true
}
