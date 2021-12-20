package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type SignInRequest struct {
	Address string
}
type SignInResponse struct {
	Address string
}

func signinWrapper(w http.ResponseWriter, r *http.Request) {
	var incoming SignInRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&incoming); err != nil {
		http.Error(w, fmt.Sprintf("invalid request: %s", err), http.StatusBadRequest)
		return
	}
	log.Printf("[SIGNIN] parsed %q", incoming)
	resp, err := signinHandler(incoming)
	if err != nil {
		http.Error(w, fmt.Sprintf("coudl not sign in: %s", err), http.StatusBadRequest)
		return
	}
	outgoing, err := json.Marshal(resp)
	if err != nil {
		log.Fatalf("could not serialize response: %s", err)
	}
	if _, err := w.Write(outgoing); err != nil {
		log.Fatalf("could not write response: %s", err)
	}
}
func signinHandler(req SignInRequest) (SignInResponse, error) {
	return SignInResponse{}, nil
}

func main() {
	const address = "localhost:8080"
	http.HandleFunc("/signin", signinWrapper)
	log.Printf("listening on %s...\n", address)
	log.Fatal(http.ListenAndServe(address, nil))
}
