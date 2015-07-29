package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/srhnsn/securefilearchiver/models"
	"github.com/srhnsn/securefilearchiver/utils"
)

const (
	databaseFilename        = "index.json"
	unusedChunksDeleteBatch = "delete unused chunks.bat"
)

type chunkIndexMap map[string]bool
type removedPathsMap map[string]bool

func createUnusedChunksDeleteBatch(files []string, directory string) {
	if len(files) == 0 {
		return
	}

	batchPath := filepath.Join(directory, unusedChunksDeleteBatch)

	out := []string{"@echo off", ""}

	for _, file := range files {
		out = append(out, getDeleteCmd(file))
	}

	out = append(out, "")
	out = append(out, getDeleteCmd(unusedChunksDeleteBatch))
	out = append(out, "")
	out = append(out, "pause")

	data := []byte(strings.Join(out, "\r\n"))
	err := ioutil.WriteFile(batchPath, data, 0700)

	if err != nil {
		utils.Error.Panicln(err)
	}
}

func decryptIndexKey(doc *models.Document, password string) {
	data, err := hex.DecodeString(doc.KeyEncrypted)

	if err != nil {
		utils.Error.Panicln(err)
	}

	doc.KeyUnencrypted = string(utils.DecryptData(data, password))
}

func encryptIndexKey(doc *models.Document, password string) {
	key := utils.EncryptData([]byte(doc.KeyUnencrypted), password)
	doc.KeyEncrypted = hex.EncodeToString(key)
}

func getChunkIndexMap(doc *models.Document) chunkIndexMap {
	chunkIndex := chunkIndexMap{}

	for _, file := range doc.Files {
		for _, chunk := range file.Chunks {
			chunkIndex[chunk.Name] = true
		}
	}

	for _, fileVersions := range doc.DeletedFiles {
		for _, file := range fileVersions {
			for _, chunk := range file.Chunks {
				chunkIndex[chunk.Name] = true
			}
		}
	}

	return chunkIndex
}

func getExistingIndexFilename(directory string) string {
	base := filepath.Join(directory, databaseFilename)

	if utils.FileExists(base) {
		return base
	}

	if utils.FileExists(base + ZipSuffix) {
		return base + ZipSuffix
	}

	if utils.FileExists(base + EncSuffix) {
		return base + EncSuffix
	}

	if utils.FileExists(base + ZipSuffix + EncSuffix) {
		return base + ZipSuffix + EncSuffix
	}

	return getIndexFilename(directory)
}

func getIndexFilename(directory string) string {
	filename := filepath.Join(directory, databaseFilename)

	if !*noIndexZip {
		filename += ZipSuffix
	}

	if !*noIndexEnc {
		filename += EncSuffix
	}

	return filename
}

func getNewDocument() *models.Document {
	keyUnencrypted := utils.GetNewOpenSSLKey()

	return &models.Document{
		KeyUnencrypted: keyUnencrypted,
		Files:          map[string]models.File{},
		DeletedFiles:   map[string][]models.File{},
	}
}

func getUnusedChunks(chunkIndex chunkIndexMap, directory string) []string {
	unusedChunks := []string{}

	encryptedIndexName := databaseFilename + EncSuffix

	walkFn := func(fullPath string, fileInfo os.FileInfo, err error) error {
		if fileInfo.IsDir() {
			return nil
		}

		filename := fileInfo.Name()

		if !strings.HasSuffix(filename, EncSuffix) {
			return nil
		}

		if filename == encryptedIndexName {
			return nil
		}

		chunkName := filename[:len(filename)-len(EncSuffix)]

		_, exists := chunkIndex[chunkName]

		if exists {
			return nil
		}

		relativePath, err := filepath.Rel(directory, fullPath)

		if err != nil {
			utils.Error.Panicln(err)
		}

		unusedChunks = append(unusedChunks, relativePath)

		return nil
	}

	filepath.Walk(directory, walkFn)

	return unusedChunks
}

func getRemovedPathsMap(doc *models.Document) removedPathsMap {
	paths := removedPathsMap{}

	for path := range doc.Files {
		paths[path] = true
	}

	return paths
}

func readIndex(filename string) (*models.Document, error) {
	if !utils.FileExists(filename) {
		utils.Info.Printf("no index found at %s, creating new archive\n", filename)

		return getNewDocument(), nil
	}

	data, err := ioutil.ReadFile(filename)

	if err != nil {
		return nil, err
	}

	data = unpackIndex(data, filename)

	var document models.Document

	err = json.Unmarshal(data, &document)

	if err != nil {
		return nil, err
	}

	decryptIndexKey(&document, getPassword())

	return &document, nil
}

func saveIndex(filename string, doc *models.Document) {
	encryptIndexKey(doc, getPassword())
	data, err := json.MarshalIndent(doc, "", "\t")

	if err != nil {
		utils.Error.Panicln(err)
	}

	if !*noIndexZip {
		data = utils.CompressData(data)
	}

	if !*noIndexEnc {
		data = utils.EncryptData(data, getPassword())
	}

	tempFilename := filename + TmpSuffix
	err = ioutil.WriteFile(tempFilename, data, 0700)

	if err != nil {
		utils.Error.Panicln(err)
	}

	utils.Trace.Println("validating index")
	err = validateIndex(tempFilename, doc)

	if err != nil {
		utils.Error.Panicln(err)
	}

	// Ignore the error. Remove line when Go 1.5 arrives.
	// See https://github.com/golang/go/issues/8914#issuecomment-99570437
	// See https://github.com/golang/go/issues/8914#issuecomment-99570437
	os.Remove(filename)
	err = os.Rename(tempFilename, filename)

	if err != nil {
		utils.Error.Panicln(err)
	}
}

func unpackIndex(data []byte, filename string) []byte {
	if strings.HasSuffix(filename, TmpSuffix) {
		// Strip TmpSuffix
		filename = filename[:len(filename)-len(TmpSuffix)]
	}

	if strings.HasSuffix(filename, EncSuffix) {
		// Strip EncSuffix and decrypt
		filename = filename[:len(filename)-len(EncSuffix)]
		data = utils.DecryptData(data, getPassword())
	}

	if strings.HasSuffix(filename, ZipSuffix) {
		data = utils.UncompressData(data)
	}

	return data
}

func validateIndex(filename string, oldDoc *models.Document) error {
	doc, err := readIndex(filename)

	if err != nil {
		return err
	}

	if len(doc.Files) != len(oldDoc.Files) {
		return fmt.Errorf("lengths of doc.Files (%d) and oldDoc.Files (%d) are not equal", len(doc.Files), len(oldDoc.Files))
	}

	if len(doc.DeletedFiles) != len(oldDoc.DeletedFiles) {
		return fmt.Errorf("lengths of doc.DeletedFiles (%d) and oldDoc.DeletedFiles (%d) are not equal", len(doc.DeletedFiles), len(oldDoc.DeletedFiles))
	}

	return nil
}
