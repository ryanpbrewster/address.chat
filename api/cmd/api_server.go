package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

type ChallengeRequest struct {
	Address string
}
type ChallengeResponse struct {
	Challenge string
}

func challengeWrapper(w http.ResponseWriter, r *http.Request) {
	log.Printf("recv %q @ %q", r.Method, r.URL.Path)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	var incoming ChallengeRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&incoming); err != nil {
		http.Error(w, fmt.Sprintf("invalid request: %s", err), http.StatusBadRequest)
		return
	}
	log.Printf("[AUTH/CHALLENGE] parsed %q", incoming)
	resp := ChallengeResponse{Challenge: fmt.Sprintf("it is %s", time.Now().UTC())}
	outgoing, err := json.Marshal(resp)
	if err != nil {
		log.Fatalf("could not serialize response: %s", err)
	}
	if _, err := w.Write(outgoing); err != nil {
		log.Fatalf("could not write response: %s", err)
	}
}

type SignInRequest struct {
	Address   string
	Challenge string
	Signature string
}
type SignInResponse struct {
	Token string
}

func signinWrapper(w http.ResponseWriter, r *http.Request) {
	log.Printf("recv %q @ %q", r.Method, r.URL.Path)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	var incoming SignInRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&incoming); err != nil {
		http.Error(w, fmt.Sprintf("invalid request: %s", err), http.StatusBadRequest)
		return
	}
	log.Printf("[AUTH/SIGNIN] parsed %q", incoming)
	token, err := verifySignature(incoming.Address, incoming.Challenge, incoming.Signature)
	if err != nil {
		http.Error(w, fmt.Sprintf("unknown error: %s", err), http.StatusInternalServerError)
		return
	}
	outgoing, err := json.Marshal(SignInResponse{Token: token})
	if err != nil {
		log.Fatalf("could not serialize response: %s", err)
	}
	if _, err := w.Write(outgoing); err != nil {
		log.Fatalf("could not write response: %s", err)
	}
}
func verifySignature(address string, challenge string, signature string) (string, error) {
	log.Printf("trying to verify signature from %s on challenge '%s' = '%s'", address, challenge, signature)

	sig, err := hexutil.Decode(signature)
	if err != nil {
		return "", fmt.Errorf("invalid signature: %s", err)
	}
	// https://github.com/ethereum/go-ethereum/blob/55599ee95d4151a2502465e0afc7c47bd1acba77/internal/ethapi/api.go#L442
	if sig[64] != 27 && sig[64] != 28 {
		return "", fmt.Errorf("magic bytes are off somehow?")
	}
	sig[64] -= 27

	pubKey, err := crypto.SigToPub(signHash([]byte(challenge)), sig)
	if err != nil {
		return "", fmt.Errorf("could not verify signature: %s", err)
	}

	if crypto.PubkeyToAddress(*pubKey) != common.HexToAddress(address) {
		return "", fmt.Errorf("signature does not proove ownership for %s", address)
	}
	return "put token here", nil
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

func main() {
	const address = "localhost:8080"
	http.HandleFunc("/auth/challenge", challengeWrapper)
	http.HandleFunc("/auth/signin", signinWrapper)
	log.Printf("listening on %s...\n", address)
	log.Fatal(http.ListenAndServe(address, nil))
}
