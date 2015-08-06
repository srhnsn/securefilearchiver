package utils

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
	"golang.org/x/crypto/openpgp/packet"
)

const (
	gnupgBinary = "gpg2"
)

var (
	encryptionConfig = &packet.Config{
		DefaultCipher: packet.CipherAES256,
	}
)

// DecryptData decrypts data using OpenPGP decryption.
func DecryptData(input []byte, password string) []byte {
	inputReader := bytes.NewReader(input)

	tried := false

	md, err := openpgp.ReadMessage(inputReader, nil, func(keys []openpgp.Key, symmetric bool) ([]byte, error) {
		if tried {
			return nil, errors.New("invalid password")
		}

		tried = true
		return []byte(password), nil
	}, nil)

	PanicIfErr(err)

	output, err := ioutil.ReadAll(md.UnverifiedBody)

	PanicIfErr(err)

	return output
}

// DecryptDataArmored decrypts armored data using OpenPGP decryption.
func DecryptDataArmored(input []byte, password string) []byte {
	inputReader := bytes.NewReader(input)
	block, err := armor.Decode(inputReader)

	PanicIfErr(err)

	armorReader := block.Body
	unarmoredInput, err := ioutil.ReadAll(armorReader)

	PanicIfErr(err)

	return DecryptData(unarmoredInput, password)
}

// EncryptData encrypts data using symmetric OpenPGP encryption.
func EncryptData(input []byte, password string) []byte {
	var output bytes.Buffer

	cryptoWriter, err := openpgp.SymmetricallyEncrypt(&output, []byte(password), nil, encryptionConfig)
	PanicIfErr(err)

	_, err = cryptoWriter.Write(input)
	PanicIfErr(err)

	err = cryptoWriter.Close()
	PanicIfErr(err)

	return output.Bytes()
}

// EncryptDataArmored encrypts data using symmetric OpenPGP encryption.
// The result will be armored OpenPGP output.
func EncryptDataArmored(input []byte, password string) []byte {
	var output bytes.Buffer
	encryptedInput := EncryptData(input, password)

	armorWriter, err := armor.Encode(&output, "PGP MESSAGE", nil)
	PanicIfErr(err)

	_, err = armorWriter.Write(encryptedInput)
	PanicIfErr(err)

	err = armorWriter.Close()
	PanicIfErr(err)

	return output.Bytes()
}

// GetDecryptCommand returns a Windows console command to decrypt a specific
// file that was encrypted with OpenPGP.
func GetDecryptCommand(inputFile string, outputFile string, passwordFile string) string {
	return fmt.Sprintf(`call %s --batch --decrypt --passphrase-file "%s" --quiet --output "%s" "%s"`,
		gnupgBinary,
		passwordFile,
		outputFile,
		inputFile,
	)
}

// GetHashSum returns the hash sum for data using the preferred algorithm
// (currently SHA-256).
func GetHashSum(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// GetNewDocumentKey returns 32 random bytes, encoded as a 64 byte hex string.
func GetNewDocumentKey() string {
	return getRandomHexBytes(32)
}

func getRandomHexBytes(length int) string {
	data := make([]byte, length)
	_, err := io.ReadFull(rand.Reader, data)

	PanicIfErr(err)

	return hex.EncodeToString(data)
}
