package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"address.chat/api/auth"
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
	if err := auth.VerifySignature(incoming.Address, incoming.Challenge, incoming.Signature); err != nil {
		http.Error(w, fmt.Sprintf("unknown error: %s", err), http.StatusInternalServerError)
		return
	}
	outgoing, err := json.Marshal(SignInResponse{Token: "put token here"})
	if err != nil {
		log.Fatalf("could not serialize response: %s", err)
	}
	if _, err := w.Write(outgoing); err != nil {
		log.Fatalf("could not write response: %s", err)
	}
}

func main() {
	const address = "localhost:8080"
	http.HandleFunc("/auth/challenge", challengeWrapper)
	http.HandleFunc("/auth/signin", signinWrapper)
	log.Printf("listening on %s...\n", address)
	log.Fatal(http.ListenAndServe(address, nil))
}
