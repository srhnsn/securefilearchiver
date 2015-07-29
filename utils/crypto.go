package utils

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"io"
	"os"
	"os/exec"
)

// PasswordEnv is the environment variable that is used for password
// storage for OpenSSL.
const PasswordEnv = "SFA_PASSWORD"

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

// DecryptDataEnvPassword decrypts data using symmetric OpenSSL decryption.
// It uses environment variables for passing the password.
func DecryptDataEnvPassword(data []byte, password string) []byte {
	openSSLPassword := "env:" + PasswordEnv

	err := os.Setenv(PasswordEnv, password)

	if err != nil {
		Error.Panicln(err)
	}

	result := DecryptData(data, openSSLPassword)

	err = os.Setenv(PasswordEnv, "")

	if err != nil {
		Error.Panicln(err)
	}

	return result
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

// EncryptDataEnvPassword encrypts data using symmetric OpenSSL encryption.
// It uses environment variables for passing the password.
func EncryptDataEnvPassword(data []byte, password string) []byte {
	openSSLPassword := "env:" + PasswordEnv

	err := os.Setenv(PasswordEnv, password)

	if err != nil {
		Error.Panicln(err)
	}

	result := EncryptData(data, openSSLPassword)

	err = os.Setenv(PasswordEnv, "")

	if err != nil {
		Error.Panicln(err)
	}

	return result
}

// GetNewOpenSSLKey returns 32 random bytes, encoded as a 64 byte hex string.
func GetNewOpenSSLKey() string {
	return getRandomHexBytes(32)
}

func getRandomHexBytes(length int) string {
	data := make([]byte, length)
	_, err := io.ReadFull(rand.Reader, data)

	if err != nil {
		Error.Panicln(err)
	}

	return hex.EncodeToString(data)
}

func getSaltHex() string {
	return getRandomHexBytes(8)
}

func runOpenSSLCommand(cmd *exec.Cmd, input []byte) []byte {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdin = bytes.NewReader(input)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	if err != nil {
		Error.Panicf("OpenSSL error: %s. Stderr: %s", err, stderr.String())
	}

	if stderr.Len() != 0 {
		Error.Panicf("OpenSSL stderr not empty: %s", stderr.String())
	}

	if stdout.Len() == 0 {
		Error.Panicf("OpenSSL stdout empty")
	}

	return stdout.Bytes()
}
