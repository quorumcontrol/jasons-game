package trees

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	types "github.com/golang/protobuf/ptypes"
	anypb "github.com/golang/protobuf/ptypes/any"
	cbor "github.com/ipfs/go-ipld-cbor"
	"github.com/pkg/errors"
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

	any, err := types.MarshalAny(i)
	if err != nil {
		return errors.Wrap(err, "error turning into any")
	}

	// marshalling a protobuf any doesn't store the underlying
	// type as cbor, so make a any-like map and deal with type
	// and value manually
	toStore := map[string]interface{}{
		"typeUrl": any.GetTypeUrl(),
		"value":   i,
	}
	return tree.updatePath([]string{"interactions", i.GetCommand()}, toStore)
}

func (w *withInteractions) getInteractionFromTree(tree updatableTree, command string) (Interaction, error) {
	val, err := tree.getPath([]string{"interactions", command})
	if err != nil || val == nil {
		return nil, err
	}

	typeURL, ok := val.(map[string]interface{})["typeUrl"]
	if !ok || typeURL.(string) == "" {
		return nil, fmt.Errorf("interaction was not stored with protobuf typeUrl")
	}

	interactionValue, ok := val.(map[string]interface{})["value"]
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
