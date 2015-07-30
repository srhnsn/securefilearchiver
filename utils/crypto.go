package utils

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os/exec"
)

const gnupgBinary = "gpg2"

// DecryptData decrypts data using GnuPG decryption.
func DecryptData(input []byte, password string) []byte {
	cmd := exec.Command(
		gnupgBinary,
		"--batch",
		"--decrypt",
		"--passphrase-fd", "0",
		"--quiet",
	)

	input = append([]byte(password+"\n"), input...)

	return runGnuPGCommand(cmd, input)
}

// EncryptData encrypts data using symmetric GnuPG encryption.
func EncryptData(input []byte, password string) []byte {
	cmd := exec.Command(
		gnupgBinary,
		"--batch",
		"--cipher-algo", "AES-256",
		"--compress-algo", "none",
		"--force-mdc",
		"--passphrase-fd", "0",
		"--symmetric",
	)

	input = append([]byte(password+"\n"), input...)

	return runGnuPGCommand(cmd, input)
}

// EncryptDataArmored encrypts data using symmetric GnuPG encryption.
// The result will be armored GnuPG output.
func EncryptDataArmored(input []byte, password string) []byte {
	cmd := exec.Command(
		gnupgBinary,
		"--armor",
		"--batch",
		"--cipher-algo", "AES-256",
		"--compress-algo", "none",
		"--force-mdc",
		"--passphrase-fd", "0",
		"--symmetric",
	)

	input = append([]byte(password+"\n"), input...)

	return runGnuPGCommand(cmd, input)
}

// GetDecryptCommand returns a Windows console command to decrypt a specific
// file that was encrypted with GnuPG
func GetDecryptCommand(inputFile string, outputFile string, password string) string {
	return fmt.Sprintf(`echo %s| %s --batch --decrypt --passphrase-fd 0 --quiet --output "%s" "%s"`,
		password,
		gnupgBinary,
		outputFile,
		inputFile,
	)
}

// GetNewDocumentKey returns 32 random bytes, encoded as a 64 byte hex string.
func GetNewDocumentKey() string {
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

func runGnuPGCommand(cmd *exec.Cmd, input []byte) []byte {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdin = bytes.NewReader(input)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	if err != nil {
		Error.Panicf("GnuPG error: %s. Stderr: %s", err, stderr.String())
	}

	if stderr.Len() != 0 {
		Error.Panicf("GnuPG stderr not empty: %s", stderr.String())
	}

	if stdout.Len() == 0 {
		Error.Panicf("GnuPG stdout empty")
	}

	return stdout.Bytes()
}
