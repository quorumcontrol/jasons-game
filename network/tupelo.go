package network

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"

	"github.com/google/uuid"
	cid "github.com/ipfs/go-cid"
	"github.com/pkg/errors"
	"github.com/quorumcontrol/chaintree/chaintree"
	"github.com/quorumcontrol/chaintree/nodestore"
	"github.com/quorumcontrol/messages/build/go/signatures"
	"github.com/quorumcontrol/messages/build/go/transactions"
	"github.com/quorumcontrol/tupelo-go-sdk/client"
	"github.com/quorumcontrol/tupelo-go-sdk/consensus"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/remote"
	"github.com/quorumcontrol/tupelo-go-sdk/gossip3/types"
)

type Tupelo struct {
	Store        nodestore.DagStore
	NotaryGroup  *types.NotaryGroup
	PubSubSystem remote.PubSub
}

func (t *Tupelo) SubscribeToCurrentStateChanges(did string, fn func(msg *signatures.CurrentState)) (func(), error) {
	cli := client.New(t.NotaryGroup, did, t.PubSubSystem)
	cli.Listen()
	sub, err := cli.SubscribeAll(fn)

	if err != nil {
		return nil, err
	}

	return func() {
		cli.Unsubscribe(sub)
		cli.Stop()
	}, nil
}

func (t *Tupelo) GetTip(did string) (cid.Cid, error) {
	cli := client.New(t.NotaryGroup, did, t.PubSubSystem)
	cli.Listen()
	defer cli.Stop()

	currState, err := cli.TipRequest()
	if err != nil {
		return cid.Undef, errors.Wrap(err, "error getting tip")
	}
	if currState.Signature == nil {
		return cid.Undef, nil
	}

	tip, err := cid.Cast(currState.Signature.NewTip)
	if err != nil {
		return cid.Undef, errors.Wrap(err, "error casting tip")
	}
	return tip, nil
}

func (t *Tupelo) CreateChainTree(key *ecdsa.PrivateKey) (*consensus.SignedChainTree, error) {
	tree, err := consensus.NewSignedChainTree(key.PublicKey, t.Store)
	if err != nil {
		return nil, errors.Wrap(err, "error creating new signed chaintree")
	}

	c := client.New(t.NotaryGroup, tree.MustId(), t.PubSubSystem)
	c.Listen()
	defer c.Stop()

	return tree, nil
}

func (t *Tupelo) SendInk(source *consensus.SignedChainTree, key *ecdsa.PrivateKey, amount uint64, destinationChainId string) (*transactions.TokenPayload, error) {
	c := client.New(t.NotaryGroup, source.MustId(), t.PubSubSystem)
	c.Listen()
	defer c.Stop()

	return t.sendInk(c, source, key, amount, destinationChainId)
}

func (t *Tupelo) sendInk(c *client.Client, source *consensus.SignedChainTree, key *ecdsa.PrivateKey, amount uint64, destinationChainId string) (*transactions.TokenPayload, error) {
	txId, err := uuid.NewRandom()
	if err != nil {
		return nil, errors.Wrap(err, "error generating send ink transaction ID")
	}

	txIdString := txId.String()

	transaction, err := t.sendInkTransaction(txIdString, destinationChainId, amount)
	if err != nil {
		return nil, errors.Wrap(err, "error generating ink send token transaction")
	}

	log.Debugf("send ink transaction: %+v", *transaction)

	txResp, err := t.playTransactions(c, source, key, []*transactions.Transaction{transaction})
	if err != nil {
		return nil, errors.Wrap(err, "error playing ink send token transaction")
	}

	tokenNameString := t.NotaryGroup.Config().TransactionToken
	tokenName := consensus.TokenNameFromString(tokenNameString)
	tokenPayload, err := t.TokenPayloadForTransaction(source, &tokenName, txIdString, &txResp.Signature)
	if err != nil {
		return nil, errors.Wrap(err, "error building token payload")
	}

	return tokenPayload, nil
}

func (t *Tupelo) ReceiveInk(tree *consensus.SignedChainTree, key *ecdsa.PrivateKey, tokenPayload *transactions.TokenPayload) error {
	c := client.New(t.NotaryGroup, tree.MustId(), t.PubSubSystem)
	c.Listen()
	defer c.Stop()

	return t.receiveInk(c, tree, key, tokenPayload)
}

func (t *Tupelo) receiveInk(c *client.Client, tree *consensus.SignedChainTree, privateKey *ecdsa.PrivateKey, tokenPayload *transactions.TokenPayload) error {
	transaction, err := t.receiveInkTransaction(tree, privateKey, tokenPayload)
	if err != nil {
		return errors.Wrap(err, "error generating ink receive token transaction")
	}

	_, err = t.playTransactions(c, tree, privateKey, []*transactions.Transaction{transaction})
	if err != nil {
		return errors.Wrap(err, "error playing ink receive token transaction")
	}

	return nil
}

func (t *Tupelo) depositInk(c *client.Client, key *ecdsa.PrivateKey, tree *consensus.SignedChainTree, inkHolder *consensus.SignedChainTree) (*transactions.Transaction, error) {
	depositPayload, err := t.sendInk(c, inkHolder, key, t.NotaryGroup.Config().BurnAmount, tree.MustId())
	if err != nil {
		return nil, errors.Wrap(err, "error sending ink")
	}

	return t.receiveInkTransaction(tree, key, depositPayload)
}

func (t *Tupelo) PlayTransactions(tree *consensus.SignedChainTree, key *ecdsa.PrivateKey, transactions []*transactions.Transaction, inkHolder *consensus.SignedChainTree) (*consensus.AddBlockResponse, error) {
	c := client.New(t.NotaryGroup, tree.MustId(), t.PubSubSystem)
	c.Listen()
	defer c.Stop()

	if t.shouldBurn() {
		depositTx, err := t.depositInk(c, key, tree, inkHolder)
		if err != nil {
			return nil, errors.Wrap(err, "error depositing ink")
		}

		txnsWithDeposit := append(transactions, depositTx)
		return t.playTransactions(c, tree, key, txnsWithDeposit)
	}

	return t.playTransactions(c, tree, key, transactions)
}

func (t *Tupelo) playTransactions(c *client.Client, tree *consensus.SignedChainTree, key *ecdsa.PrivateKey, transactions []*transactions.Transaction) (*consensus.AddBlockResponse, error) {
	var tipPtr *cid.Cid
	if !tree.IsGenesis() {
		tip := tree.Tip()
		tipPtr = &tip
	}

	if t.shouldBurn() {
		burnTransaction, err := t.inkBurnTransaction(tree)
		if err != nil {
			return nil, errors.Wrap(err, "error generating ink burn transaction")
		}

		return c.PlayTransactions(tree, key, tipPtr, append(transactions, burnTransaction))
	}

	return c.PlayTransactions(tree, key, tipPtr, transactions)
}

func (t *Tupelo) shouldBurn() bool {
	return t.NotaryGroup.Config().BurnAmount > 0 && t.NotaryGroup.Config().TransactionToken != ""
}

func (t *Tupelo) inkBurnTransaction(sourceTree *consensus.SignedChainTree) (*transactions.Transaction, error) {
	tipString := sourceTree.Tip().String()

	txIdBytes := sha256.Sum256([]byte(tipString + "burn"))
	txId := hex.EncodeToString(txIdBytes[:])

	burnTx, err := t.sendInkTransaction(txId, "", t.NotaryGroup.Config().BurnAmount)
	if err != nil {
		return nil, errors.Wrap(err, "error generating ink burn transaction")
	}

	return burnTx, nil
}

func (t *Tupelo) sendInkTransaction(txId string, destId string, amount uint64) (*transactions.Transaction, error) {
	transaction, err := chaintree.NewSendTokenTransaction(txId, t.NotaryGroup.Config().TransactionToken, amount, destId)
	if err != nil {
		return nil, errors.Wrap(err, "error generating send ink transaction")
	}

	return transaction, nil
}

func (t *Tupelo) receiveInkTransaction(tree *consensus.SignedChainTree, key *ecdsa.PrivateKey, tokenPayload *transactions.TokenPayload) (*transactions.Transaction, error) {
	decodedTip, err := cid.Decode(tokenPayload.Tip)
	if err != nil {
		return nil, errors.Wrapf(err, "error decoding token payload tip: %s", tokenPayload.Tip)
	}

	log.Debugf("receive ink decoded token payload tip: %s", decodedTip)

	return chaintree.NewReceiveTokenTransaction(tokenPayload.TransactionId, decodedTip.Bytes(), tokenPayload.Signature, tokenPayload.Leaves)

}

func (t *Tupelo) TokenPayloadForTransaction(tree *consensus.SignedChainTree, tokenName *consensus.TokenName, sendTokenTxId string, sendTxSig *signatures.Signature) (*transactions.TokenPayload, error) {
	c := client.New(t.NotaryGroup, tree.MustId(), t.PubSubSystem)
	c.Listen()
	defer c.Stop()

	return c.TokenPayloadForTransaction(tree.ChainTree, tokenName, sendTokenTxId, sendTxSig)
}
