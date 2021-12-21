package auth

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

func VerifySignature(address string, challenge string, signature string) error {
	sig, err := hexutil.Decode(signature)
	if err != nil {
		return fmt.Errorf("invalid signature: %s", err)
	}
	if len(sig) != 65 {
		return fmt.Errorf("expected signature to be %q bytes, got %q bytes", 65, len(sig))
	}
	// https://github.com/ethereum/go-ethereum/blob/55599ee95d4151a2502465e0afc7c47bd1acba77/internal/ethapi/api.go#L442
	if sig[64] != 27 && sig[64] != 28 {
		return fmt.Errorf("magic bytes are off somehow?")
	}
	sig[64] -= 27

	pubKey, err := crypto.SigToPub(signHash([]byte(challenge)), sig)
	if err != nil {
		return fmt.Errorf("could not verify signature: %s", err)
	}

	if crypto.PubkeyToAddress(*pubKey) != common.HexToAddress(address) {
		return fmt.Errorf("signature does not proove ownership for %s", address)
	}
	return nil
}

// https://github.com/ethereum/go-ethereum/blob/55599ee95d4151a2502465e0afc7c47bd1acba77/internal/ethapi/api.go#L404
// signHash is a helper function that calculates a hash for the given message that can be
// safely used to calculate a signature from.
//
// The hash is calculated as
//   keccak256("\x19Ethereum Signed Message:\n"${message length}${message}).
//
// This gives context to the signed message and prevents signing of transactions.
func signHash(data []byte) []byte {
	msg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(data), data)
	return crypto.Keccak256([]byte(msg))
}
