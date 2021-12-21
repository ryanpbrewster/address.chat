package auth

import (
	"testing"
)

func TestVerifySignature_TinyInputsDoNotPanic(t *testing.T) {
	if err := VerifySignature("0x", "", "0x"); err == nil {
		t.Errorf("expected VerifySignature('', '', '') to yield an error")
	}
}

func TestVerifySignature_ValidSignature(t *testing.T) {
	address := "0x33a8122f5c41eee796de9da8d63af7670f310964"
	challenge := "it is 2021-12-21 20:39:56.673319933 +0000 UTC"
	signature := "0x04c257dea26031415f48776068a0549d6acce6b28c9095bba02e4d82757c3b944db2c94774ef6a63cd6374a49652c60e380f2fd681b65f2f2423e6bc6c3d67d61b"
	if err := VerifySignature(address, challenge, signature); err != nil {
		t.Errorf("want nil, got VerifySignature(%q, %q, %q) = %q", address, challenge, signature, err)
	}
}

func TestVerifySignature_InvalidSignature(t *testing.T) {
	address := "0x33a8122f5c41eee796de9da8d63af7670f310964"
	challenge := "it is 2021-12-21 20:39:56.673319933 +0000 UTC"
	signatures := []string{
		// 0x14... instead of the expected 0x04...
		"0x14c257dea26031415f48776068a0549d6acce6b28c9095bba02e4d82757c3b944db2c94774ef6a63cd6374a49652c60e380f2fd681b65f2f2423e6bc6c3d67d61b",
		// Valid signature with extra bytes tacked on at the end.
		"0x04c257dea26031415f48776068a0549d6acce6b28c9095bba02e4d82757c3b944db2c94774ef6a63cd6374a49652c60e380f2fd681b65f2f2423e6bc6c3d67d61b0000",
		// Too short.
		"0x00",
		// Totally busted.
		"0x0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
	}
	for _, signature := range signatures {
		if err := VerifySignature(address, challenge, signature); err == nil {
			t.Errorf("expected an error from VerifySignature(%q, %q, %q)", address, challenge, signature)
		}
	}
}
