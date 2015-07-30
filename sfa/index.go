package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	doc.KeyUnencrypted = string(utils.DecryptData([]byte(doc.KeyEncrypted), password))
}

func encryptIndexKey(doc *models.Document, password string) {
	key := utils.EncryptDataArmored([]byte(doc.KeyUnencrypted), password)
	doc.KeyEncrypted = string(key)
}

func garbageCollect(inputDir string) {
	doc, err := readIndex(getExistingIndexFilename(inputDir))

	if err != nil {
		utils.Error.Panicln(err)
	}

	utils.Trace.Println("checking for unused chunks")
	chunkIndex := getChunkIndexMap(doc)
	unusedChunks := getUnusedChunks(chunkIndex, inputDir)
	createUnusedChunksDeleteBatch(unusedChunks, inputDir)

	if len(unusedChunks) > 0 {
		utils.Info.Printf("found %d unused chunks", len(unusedChunks))
	} else {
		utils.Info.Printf("no unused chunks")
	}
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
	keyUnencrypted := utils.GetNewDocumentKey()

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

func pruneFiles(inputDir string, pruneRangeStr string) {
	doc, err := readIndex(getExistingIndexFilename(inputDir))

	if err != nil {
		utils.Error.Panicln(err)
	}

	pruneRange, err := utils.ParseHumanRange(pruneRangeStr)

	if err != nil {
		utils.Error.Panicln(err)
	}

	pruneThreshold := time.Now().Add(-pruneRange)

	utils.Info.Printf("pruning files that were found to be deleted before %s (%s ago)\n",
		pruneThreshold, pruneRange)

	var prunedFiles uint64

	for shortPath, versions := range doc.DeletedFiles {
		newVersions := []models.File{}

		for _, file := range versions {
			if file.DeletedAt.Time.After(pruneThreshold) {
				newVersions = append(newVersions, file)
				continue
			}

			prunedFiles++
		}

		if len(newVersions) > 0 {
			doc.DeletedFiles[shortPath] = newVersions
		} else {
			delete(doc.DeletedFiles, shortPath)
		}
	}

	utils.Trace.Printf("pruned %d files\n", prunedFiles)

	utils.Trace.Println("writing to index")
	saveIndex(getIndexFilename(inputDir), doc)
}

func readIndex(filename string) (*models.Document, error) {
	utils.Trace.Println("reading index")

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
