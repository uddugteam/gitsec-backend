package signer

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// Signer is a structure of ETH account
// with private key and an address
type Signer struct {
	Address common.Address
	private *ecdsa.PrivateKey

	gasLimit uint
}

// NewSigner create new Signer instance from
// given private key string
func NewSigner(privateKeyString string, gasLimit uint) (*Signer, error) {
	privateKey, err := crypto.HexToECDSA(privateKeyString)
	if err != nil {
		return nil, fmt.Errorf("crete ECDSA private key from given HEX string: %w", err)
	}

	publicKeyECDSA, ok := privateKey.Public().(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("error casting public key to ECDSA")
	}

	return &Signer{
		Address:  crypto.PubkeyToAddress(*publicKeyECDSA),
		private:  privateKey,
		gasLimit: 200000,
	}, nil
}

// Sign create transactor signer
func (s *Signer) Sign(chainID *big.Int) (*bind.TransactOpts, error) {
	signer, err := bind.NewKeyedTransactorWithChainID(s.private, chainID)
	if err != nil {
		return nil, fmt.Errorf("create transactor opts: %w", err)
	}

	if s.gasLimit != 0 {
		signer.GasLimit = uint64(s.gasLimit)
	}

	return signer, nil
}
