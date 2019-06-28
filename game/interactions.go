package game

import (
	"crypto/rand"
	"fmt"
	"io"

	"golang.org/x/crypto/scrypt"
	"golang.org/x/crypto/nacl/secretbox"
	"github.com/golang/protobuf/proto"
	types "github.com/golang/protobuf/ptypes"
	anypb "github.com/golang/protobuf/ptypes/any"
	cbor "github.com/ipfs/go-ipld-cbor"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/chaintree/safewrap"
	"github.com/quorumcontrol/chaintree/typecaster"
)

func init() {
	cbor.RegisterCborType(RespondInteraction{})
	typecaster.AddType(RespondInteraction{})
	cbor.RegisterCborType(ChangeLocationInteraction{})
	typecaster.AddType(ChangeLocationInteraction{})
	cbor.RegisterCborType(PickUpObjectInteraction{})
	typecaster.AddType(PickUpObjectInteraction{})
	cbor.RegisterCborType(DropObjectInteraction{})
	typecaster.AddType(DropObjectInteraction{})
	cbor.RegisterCborType(GetTreeValueInteraction{})
	typecaster.AddType(GetTreeValueInteraction{})
	cbor.RegisterCborType(CipherInteraction{})
	typecaster.AddType(CipherInteraction{})
}

type Interaction interface {
	proto.Message
	GetCommand() string
}

var _ Interaction = (*RespondInteraction)(nil)
var _ Interaction = (*ChangeLocationInteraction)(nil)
var _ Interaction = (*PickUpObjectInteraction)(nil)
var _ Interaction = (*DropObjectInteraction)(nil)
var _ Interaction = (*GetTreeValueInteraction)(nil)
var _ Interaction = (*CipherInteraction)(nil)

type ListInteractionsRequest struct{}

type ListInteractionsResponse struct {
	Interactions []Interaction
	Error        error
}

type AddInteractionRequest struct {
	Interaction Interaction
}

type AddInteractionResponse struct {
	Error error
}

type withInteractions struct {
}

type updatableTree interface {
	getPath([]string) (interface{}, error)
	updatePath([]string, interface{}) error
}

func (w *withInteractions) addInteractionToTree(tree updatableTree, i Interaction) error {
	resp, err := w.getInteractionFromTree(tree, i.GetCommand())
	if err != nil {
		return err
	}
	if resp != nil {
		return fmt.Errorf("interaction %v already exists", i.GetCommand())
	}

	toStore, err := interactionToCborNode(i)
	if err != nil {
		return errors.Wrap(err, "error turning interaction into cbor")
	}
	return tree.updatePath([]string{"interactions", i.GetCommand()}, toStore)
}

func (w *withInteractions) getInteractionFromTree(tree updatableTree, command string) (Interaction, error) {
	val, err := tree.getPath([]string{"interactions", command})
	if err != nil || val == nil {
		return nil, err
	}
	return interactionFromStoredMap(val.(map[string]interface{}))
}

func (w *withInteractions) interactionsListFromTree(tree updatableTree) ([]Interaction, error) {
	val, err := tree.getPath([]string{"interactions"})
	if err != nil || val == nil {
		return nil, err
	}

	interactions := make([]Interaction, len(val.(map[string]interface{})))
	i := 0
	for cmd := range val.(map[string]interface{}) {
		interaction, err := w.getInteractionFromTree(tree, cmd)
		if err != nil {
			return nil, err
		}
		interactions[i] = interaction
		i++
	}
	return interactions, nil
}

func interactionToCborNode(i Interaction) (*cbor.Node, error) {
	any, err := types.MarshalAny(i)
	if err != nil {
		return nil, errors.Wrap(err, "error turning into any")
	}

	// marshalling a protobuf any doesn't store the underlying
	// type as cbor, so make a any-like map and deal with type
	// and value manually
	toStore := map[string]interface{}{
		"typeUrl": any.GetTypeUrl(),
		"value":   i,
	}

	sw := safewrap.SafeWrap{}
	node := sw.WrapObject(toStore)
	return node, sw.Err
}

func interactionFromCborBytes(nodeBytes []byte) (Interaction, error) {
	sw := safewrap.SafeWrap{}
	node := sw.Decode(nodeBytes)
	if sw.Err != nil {
		return nil, errors.Wrap(sw.Err, "error decoding interaction cbor bytes")
	}
	val, _, err := node.Resolve([]string{})
	if err != nil {
		return nil, errors.Wrap(err, "error resolving interaction cbor")
	}
	return interactionFromStoredMap(val.(map[string]interface{}))
}

func interactionFromStoredMap(m map[string]interface{}) (Interaction, error) {
	typeURL, ok := m["typeUrl"]
	if !ok || typeURL.(string) == "" {
		return nil, fmt.Errorf("interaction was not stored with protobuf typeUrl")
	}

	interactionValue, ok := m["value"]
	if !ok {
		return nil, fmt.Errorf("interaction was not stored with protobuf typeUrl")
	}

	interaction, err := types.Empty(&anypb.Any{TypeUrl: typeURL.(string)})
	if err != nil {
		return nil, fmt.Errorf("protobuf type %v not found: %v", typeURL, err)
	}

	err = typecaster.ToType(interactionValue, interaction)
	if err != nil {
		return nil, errors.Wrap(err, "error casting interaction")
	}

	return interaction.(Interaction), nil
}

const cipherNonceLength = 24

func NewCipherInteraction(command string, secret string, interactionToSeal Interaction, failureInteraction Interaction) (*CipherInteraction, error) {
	if interactionToSeal.GetCommand() != "" {
		return nil, fmt.Errorf("the interactionToSeal.command must be empty - it will be autoset to command + secret")
	}

	interactionToSealNode, err := interactionToCborNode(interactionToSeal)
	if err != nil {
		return nil, errors.Wrap(err, "interactionToSeal could not be encoded")
	}
	interactionToSealBytes := interactionToSealNode.RawData()

	failureInteractionNode, err := interactionToCborNode(failureInteraction)
	if err != nil {
		return nil, errors.Wrap(err, "failureInteraction could not be encoded")
	}
	failureInteractionBytes := failureInteractionNode.RawData()

	var nonce [cipherNonceLength]byte
	if _, err = io.ReadFull(rand.Reader, nonce[:]); err != nil {
		return nil, err
	}

	cipherKey, err := cipherKey([]byte(secret), nonce[:])
	if err != nil {
		return nil, err
	}

	sealedInteractionBytes := secretbox.Seal(nonce[:], interactionToSealBytes, &nonce, &cipherKey)

	return &CipherInteraction{
		Command:                 command,
		SealedInteractionBytes:  sealedInteractionBytes,
		FailureInteractionBytes: failureInteractionBytes,
	}, nil
}

func cipherKey(secret []byte, salt []byte) (b [32]byte, err error) {
	k, err := scrypt.Key(secret, salt, 32768, 8, 1, 32)
	if err != nil {
		return b, errors.Wrap(err, "error generating secret key")
	}
	copy(b[:], k)
	return b, nil
}

func (i *CipherInteraction) Unseal(secret string) (Interaction, bool, error) {
	sealedBytes := i.SealedInteractionBytes
	var nonce [cipherNonceLength]byte
	copy(nonce[:], sealedBytes[:cipherNonceLength])

	cipherKey, err := cipherKey([]byte(secret), nonce[:])
	if err != nil {
		return nil, false, err
	}

	unsealedBytes, unsealSuccess := secretbox.Open(nil, sealedBytes[cipherNonceLength:], &nonce, &cipherKey)

	var interactionBytes []byte
	if unsealSuccess {
		interactionBytes = unsealedBytes
	} else {
		interactionBytes = i.FailureInteractionBytes
	}

	interaction, err := interactionFromCborBytes(interactionBytes)
	if err != nil {
		return nil, unsealSuccess, errors.Wrap(err, "error decoding interaction")
	}
	return interaction, unsealSuccess, nil
}
