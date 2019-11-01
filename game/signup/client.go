package signup

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/ecies"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/jasons-game/game/static"
	"github.com/quorumcontrol/jasons-game/network"
	"github.com/quorumcontrol/jasons-game/pb/jasonsgame"
)

var encryptionPubKeyPath = []string{"tree", "data", "jasons-game", "encryption-pubkey"}

type Client struct {
	network network.Network
	did     string
}

func NewClient(net network.Network) (*Client, error) {
	did, err := static.Get(net, "Signup")
	if err != nil {
		return nil, err
	}

	if did == "" {
		return nil, fmt.Errorf("Signup service has not yet been established: static.Signup is empty")
	}

	return &Client{
		network: net,
		did:     did,
	}, nil
}

func (c *Client) Did() string {
	return c.did
}

func (c *Client) Topic() []byte {
	return []byte(c.Did())
}

func (c *Client) EncryptionPubKey() (*ecdsa.PublicKey, error) {
	tree, err := c.network.GetTree(c.Did())
	if err != nil {
		return nil, errors.Wrap(err, "error finding tree")
	}

	encryptionPubKeyRaw, _, err := tree.ChainTree.Dag.Resolve(context.Background(), encryptionPubKeyPath)
	if err != nil || encryptionPubKeyRaw == nil {
		return nil, fmt.Errorf("error finding pubkey: %v", err)
	}

	encryptionPubKey, err := crypto.UnmarshalPubkey(encryptionPubKeyRaw.([]byte))
	if err != nil {
		return nil, errors.Wrap(err, "error converting pubkey")
	}

	return encryptionPubKey, nil
}

func (c *Client) Signup(email string, playerDid string) error {
	encryptionPubKey, err := c.EncryptionPubKey()
	if err != nil {
		return err
	}

	msg := &jasonsgame.SignupMessage{
		Email: email,
		Did:   playerDid,
	}

	marshaled, err := proto.Marshal(msg)
	if err != nil {
		return errors.Wrap(err, "error marshaling")
	}

	encrypted, err := ecies.Encrypt(rand.Reader, ecies.ImportECDSAPublic(encryptionPubKey), marshaled, nil, nil)
	if err != nil {
		return errors.Wrap(err, "error encryting")
	}

	encryptedMsg := &jasonsgame.SignupMessageEncrypted{
		Encrypted: encrypted,
	}

	err = c.network.Community().Send(c.Topic(), encryptedMsg)
	return errors.Wrap(err, "error sending")
}
