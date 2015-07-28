package utils

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"io"
	"os/exec"
)

// DecryptData decrypts data using symmetric OpenSSL decryption.
func DecryptData(ciphertext []byte, password string) []byte {
	cmd := exec.Command(
		"openssl",
		"enc",
		"-aes-256-cbc",
		"-d",
		"-pass",
		password,
	)

	return runOpenSSLCommand(cmd, ciphertext)
}

// EncryptData encrypts data using symmetric OpenSSL encryption.
func EncryptData(data []byte, password string) []byte {
	// Use own salt to speed up encryption.

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

	return runOpenSSLCommand(cmd, data)
}

func getSaltHex() string {
	salt := make([]byte, 8)
	_, err := io.ReadFull(rand.Reader, salt)

	if err != nil {
		Error.Fatalln(err)
	}

	return hex.EncodeToString(salt)
}

func runOpenSSLCommand(cmd *exec.Cmd, input []byte) []byte {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdin = bytes.NewReader(input)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	if err != nil {
		Error.Fatalf("OpenSSL error: %s. Stderr: %s", err, stderr.String())
	}

	if stderr.Len() != 0 {
		Error.Fatalf("OpenSSL stderr not empty: %s", stderr.String())
	}

	if stdout.Len() == 0 {
		Error.Fatalf("OpenSSL stdout empty")
	}

	return stdout.Bytes()
}
