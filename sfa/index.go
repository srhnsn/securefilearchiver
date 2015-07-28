package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/srhnsn/securefilearchiver/models"
	"github.com/srhnsn/securefilearchiver/utils"
)

const databaseFilename = "index.json"
const unusedChunksDeleteBatch = "delete unused chunks.bat"

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
		utils.Error.Fatalln(err)
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
			utils.Error.Fatalln(err)
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

		return &models.Document{
			Files:        map[string]models.File{},
			DeletedFiles: map[string][]models.File{},
		}, nil
	}

	data, err := ioutil.ReadFile(filename)

	if err != nil {
		return nil, err
	}

	var document models.Document

	err = json.Unmarshal(data, &document)

	if err != nil {
		return nil, err
	}

	return &document, nil
}

func saveIndex(filename string, doc *models.Document) {
	data, err := json.MarshalIndent(doc, "", "\t")

	if err != nil {
		utils.Error.Fatalln(err)
	}

	tempFilename := filename + ".tmp"
	err = ioutil.WriteFile(tempFilename, data, 0700)

	if err != nil {
		utils.Error.Fatalln(err)
	}

	utils.Trace.Println("validating index")
	err = validateIndex(tempFilename, doc)

	if err != nil {
		utils.Error.Fatalln(err)
	}

	// Ignore the error. Remove line when Go 1.5 arrives.
	// See https://github.com/golang/go/issues/8914#issuecomment-99570437
	os.Remove(filename)
	err = os.Rename(tempFilename, filename)

	if err != nil {
		utils.Error.Fatalln(err)
	}
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
