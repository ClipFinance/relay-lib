package signer

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
	"math/big"
)

// Signer is an interface that defines methods for signing data and transactions, and retrieving the signer's address.
type Signer interface {
	// Sign signs the given data and returns the signature.
	//
	// Parameters:
	// - data: the data to be signed.
	//
	// Returns:
	// - []byte: the signature.
	// - error: an error if the signing process fails.
	Sign(data []byte) ([]byte, error)

	// SignTx signs the given transaction with the specified chain ID and returns the signed transaction.
	//
	// Parameters:
	// - transaction: the transaction to be signed.
	// - chainID: the chain ID for the transaction.
	//
	// Returns:
	// - *ethtypes.Transaction: the signed transaction.
	// - error: an error if the signing process fails.
	SignTx(transaction *ethtypes.Transaction, chainID *big.Int) (*ethtypes.Transaction, error)

	// Address returns the signer's address.
	//
	// Returns:
	// - common.Address: the signer's address.
	Address() common.Address
}

// signer is a concrete implementation of the Signer interface.
type signer struct {
	privateKey *ecdsa.PrivateKey
	publicKey  *ecdsa.PublicKey
	address    common.Address
}

// NewSigner creates a new signer instance with the given private key.
//
// Parameters:
// - privateKey: the private key to be used for signing.
//
// Returns:
// - Signer: a new signer instance.
// - error: an error if the private key is not valid.
func NewSigner(privateKey *ecdsa.PrivateKey) (Signer, error) {
	pubKeyECDSA, ok := privateKey.Public().(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("cannot assign public key to ECDSA")
	}

	return &signer{
		privateKey: privateKey,
		publicKey:  pubKeyECDSA,
		address:    crypto.PubkeyToAddress(*pubKeyECDSA),
	}, nil
}

// Sign signs the given data and returns the signature.
//
// Parameters:
// - data: the data to be signed.
//
// Returns:
// - []byte: the signature.
// - error: an error if the signing process fails.
func (s *signer) Sign(data []byte) ([]byte, error) {
	msg := crypto.Keccak256([]byte(fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(data), data)))
	signature, err := crypto.Sign(msg, s.privateKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to sign message")
	}
	signature[64] += 27 // Transform V from 0/1 to 27/28 according to the yellow paper

	return signature, nil
}

// Address returns the signer's address.
//
// Returns:
// - common.Address: the signer's address.
func (s *signer) Address() common.Address {
	return s.address
}

// SignTx signs the given transaction with the specified chain ID and returns the signed transaction.
//
// Parameters:
// - tx: the transaction to be signed.
// - chainID: the chain ID for the transaction.
//
// Returns:
// - *ethtypes.Transaction: the signed transaction.
// - error: an error if the signing process fails.
func (s *signer) SignTx(tx *ethtypes.Transaction, chainID *big.Int) (*ethtypes.Transaction, error) {
	auth, err := bind.NewKeyedTransactorWithChainID(s.privateKey, chainID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create keyed transactor")
	}

	signedTx, err := auth.Signer(s.address, tx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to sign transaction")
	}

	return signedTx, nil
}
