package utils

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"io"
	"os/exec"
)

// DecryptData decrypts data using symmetric OpenSSL decryption.
func DecryptData(ciphertext []byte, password string) (data []byte) {
	cmd := exec.Command(
		"openssl",
		"enc",
		"-aes-256-cbc",
		"-d",
		"-pass",
		password,
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdin = bytes.NewReader(ciphertext)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	if err != nil {
		Error.Fatalf("OpenSSL decryption failed: %s. Stderr: %s", err, stderr.String())
	}

	if stderr.Len() != 0 {
		Error.Fatalf("OpenSSL stderr not empty: %s", stderr.String())
	}

	return stdout.Bytes()
}

// EncryptData encrypts data using symmetric OpenSSL decryption. It uses its
// own salt to speed up encryption.
func EncryptData(data []byte, password string) (ciphertext []byte) {
	cmd := exec.Command(
		"openssl",
		"enc",
		"-aes-256-cbc",
		"-e",
		"-S",
		getSaltHex(),
		"-pass",
		password,
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdin = bytes.NewReader(data)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	if err != nil {
		Error.Fatalf("OpenSSL encryption failed: %s. Stderr: %s", err, stderr.String())
	}

	if stderr.Len() != 0 {
		Error.Fatalf("OpenSSL stderr not empty: %s", stderr.String())
	}

	return stdout.Bytes()
}

func getSaltHex() string {
	salt := make([]byte, 8)
	_, err := io.ReadFull(rand.Reader, salt)

	if err != nil {
		Error.Fatalln(err)
	}

	return hex.EncodeToString(salt)
}
