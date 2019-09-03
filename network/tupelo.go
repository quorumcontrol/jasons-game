package network

import (
	"crypto/ecdsa"

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

func (t *Tupelo) SendInk(inkHolder *consensus.SignedChainTree, key *ecdsa.PrivateKey, amount uint64, destinationChainId string) (*transactions.TokenPayload, error) {
	c := client.New(t.NotaryGroup, inkHolder.MustId(), t.PubSubSystem)
	c.Listen()
	defer c.Stop()

	return t.sendInk(c, inkHolder, key, amount, destinationChainId)
}

func (t *Tupelo) sendInk(c *client.Client, inkHolder *consensus.SignedChainTree, key *ecdsa.PrivateKey, amount uint64, destinationChainId string) (*transactions.TokenPayload, error) {
	transaction, transactionId, err := t.sendInkTransaction(destinationChainId, amount)
	if err != nil {
		return nil, errors.Wrap(err, "error generating ink send token transaction")
	}

	log.Debugf("send ink transaction: %+v", *transaction)

	txResp, err := t.playTransactions(c, inkHolder, key, []*transactions.Transaction{transaction})
	if err != nil {
		return nil, errors.Wrap(err, "error playing ink send token transaction")
	}

	tokenNameString := t.NotaryGroup.Config().TransactionToken
	tokenName := consensus.TokenNameFromString(tokenNameString)
	tokenPayload, err := t.TokenPayloadForTransaction(inkHolder, &tokenName, transactionId.String(), &txResp.Signature)

	return tokenPayload, nil
}

func (t *Tupelo) ReceiveInkTransaction(tree *consensus.SignedChainTree, key *ecdsa.PrivateKey, tokenPayload *transactions.TokenPayload) (*transactions.Transaction, error) {
	decodedTip, err := cid.Decode(tokenPayload.Tip)
	if err != nil {
		return nil, errors.Wrapf(err, "error decoding token payload tip: %s", tokenPayload.Tip)
	}

	log.Debugf("receive ink decoded token payload tip: %s", decodedTip)

	return chaintree.NewReceiveTokenTransaction(tokenPayload.TransactionId, decodedTip.Bytes(), tokenPayload.Signature, tokenPayload.Leaves)

}

func (t *Tupelo) PlayTransactions(tree *consensus.SignedChainTree, key *ecdsa.PrivateKey, transactions []*transactions.Transaction, inkHolder *consensus.SignedChainTree) (*consensus.AddBlockResponse, error) {
	c := client.New(t.NotaryGroup, tree.MustId(), t.PubSubSystem)
	c.Listen()
	defer c.Stop()

	depositPayload, err := t.sendInk(c, inkHolder, key, t.NotaryGroup.Config().BurnAmount, tree.MustId())
	if err != nil {
		return nil, errors.Wrap(err, "error depositing ink")
	}

	depositTx, err := t.ReceiveInkTransaction(tree, key, tokenPayload)

	t.playTransactions(c, tree, key, append(transactions, depositTx))

}

func (t *Tupelo) playTransactions(c *client.Client, tree *consensus.SignedChainTree, key *ecdsa.PrivateKey, transactions []*transactions.Transaction) (*consensus.AddBlockResponse, error) {
	var tipPtr *cid.Cid
	if !tree.IsGenesis() {
		tip := tree.Tip()
		tipPtr = &tip
	}

	if t.shouldBurn() {
		burnTransaction, err := t.inkBurnTransaction()
		if err != nil {
			return nil, errors.Wrap(err, "error generating ink burn transaction")
		}

		transactions := append(transactions, burnTransaction)
	}

	return c.PlayTransactions(tree, key, tipPtr, transactions)
}

func (t *Tupelo) shouldBurn() bool {
	return t.NotaryGroup.Config().BurnAmount > 0 && t.NotaryGroup.Config().TransactionToken != ""
}

func (t *Tupelo) sendInkTransaction(destId string, amount uint64) (*transactions.Transaction, *uuid.UUID, error) {
	transactionId, err := uuid.NewRandom()
	if err != nil {
		return nil, nil, errors.Wrap(err, "error generating send ink transaction ID")
	}

	transaction, err := chaintree.NewSendTokenTransaction(transactionId.String(), t.NotaryGroup.Config().TransactionToken, amount, destId)
	if err != nil {
		return nil, nil, errors.Wrap(err, "error generating send ink transaction")
	}

	return transaction, &transactionId, nil
}

func (t *Tupelo) inkBurnTransaction() (*transactions.Transaction, error) {
	burnTx, txId, err := t.sendInkTransaction("", t.NotaryGroup.Config().BurnAmount)
	if err != nil {
		return nil, err
	}

	return burnTx, nil
}

func (t *Tupelo) TokenPayloadForTransaction(tree *consensus.SignedChainTree, tokenName *consensus.TokenName, sendTokenTxId string, sendTxSig *signatures.Signature) (*transactions.TokenPayload, error) {
	c := client.New(t.NotaryGroup, tree.MustId(), t.PubSubSystem)
	c.Listen()
	defer c.Stop()

	return c.TokenPayloadForTransaction(tree.ChainTree, tokenName, sendTokenTxId, sendTxSig)
}
