package utils

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
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

	md, err := openpgp.ReadMessage(inputReader, nil, func(keys []openpgp.Key, symmetric bool) ([]byte, error) {
		return []byte(password), nil
	}, nil)

	if err != nil {
		Error.Panicln(err)
	}

	output, err := ioutil.ReadAll(md.UnverifiedBody)

	if err != nil {
		Error.Panicln(err)
	}

	return output
}

// DecryptDataArmored decrypts armored data using OpenPGP decryption.
func DecryptDataArmored(input []byte, password string) []byte {
	inputReader := bytes.NewReader(input)
	block, err := armor.Decode(inputReader)

	if err != nil {
		Error.Panicln(err)
	}

	armorReader := block.Body
	unarmoredInput, err := ioutil.ReadAll(armorReader)

	if err != nil {
		Error.Panicln(err)
	}

	return DecryptData(unarmoredInput, password)
}

// EncryptData encrypts data using symmetric OpenPGP encryption.
func EncryptData(input []byte, password string) []byte {
	var output bytes.Buffer

	cryptoWriter, err := openpgp.SymmetricallyEncrypt(&output, []byte(password), nil, encryptionConfig)

	if err != nil {
		Error.Panicln(err)
	}

	cryptoWriter.Write(input)
	cryptoWriter.Close()

	return output.Bytes()
}

// EncryptDataArmored encrypts data using symmetric OpenPGP encryption.
// The result will be armored OpenPGP output.
func EncryptDataArmored(input []byte, password string) []byte {
	var output bytes.Buffer
	encryptedInput := EncryptData(input, password)

	armorWriter, err := armor.Encode(&output, "PGP MESSAGE", nil)

	if err != nil {
		Error.Panicln(err)
	}

	armorWriter.Write(encryptedInput)
	armorWriter.Close()

	return output.Bytes()
}

// GetDecryptCommand returns a Windows console command to decrypt a specific
// file that was encrypted with OpenPGP.
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
