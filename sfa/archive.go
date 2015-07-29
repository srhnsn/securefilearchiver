package main

import (
	"crypto/sha1"
	"encoding/hex"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/srhnsn/securefilearchiver/models"
	"github.com/srhnsn/securefilearchiver/utils"
)

// ArchiveInfo is a struct which holds all needed information for archiving
// a specific file. It is merely a convenience struct for passing in functions.
type ArchiveInfo struct {
	Document  *models.Document
	File      models.File
	FileInfo  os.FileInfo
	FileSize  uint64
	FullPath  string
	InputDir  string
	OutputDir string
	ShortPath string
}

func addToDeletedFiles(archive *ArchiveInfo) {
	archive.File.DeletedAt = models.JSONTime{Time: time.Now()}
	delete(archive.Document.Files, archive.ShortPath)

	if archive.Document.DeletedFiles == nil {
		archive.Document.DeletedFiles = map[string][]models.File{}
	}

	_, exists := archive.Document.DeletedFiles[archive.ShortPath]

	if exists {
		archive.Document.DeletedFiles[archive.ShortPath] = append(archive.Document.DeletedFiles[archive.ShortPath], archive.File)
	} else {
		archive.Document.DeletedFiles[archive.ShortPath] = []models.File{archive.File}
	}
}

func archiveFile(archive *ArchiveInfo) error {
	chunks, err := createAndGetChunks(archive)

	if err != nil {
		return err
	}

	file := models.File{
		Size:             archive.FileSize,
		ModificationTime: models.JSONTime{Time: archive.FileInfo.ModTime()},
		Chunks:           chunks,
	}

	if archive.FileInfo.IsDir() {
		file.IsDirectory = true
	}

	archive.Document.Files[archive.ShortPath] = file

	return nil
}

func chunkExists(chunkName string, archive *ArchiveInfo) bool {
	chunkPath := filepath.Join(archive.OutputDir, chunkName[:2], chunkName[:4], chunkName+EncSuffix)
	return utils.FileExists(chunkPath)
}

func createAndGetChunks(archive *ArchiveInfo) ([]models.Chunk, error) {
	if archive.FileInfo.IsDir() {
		return []models.Chunk{}, nil
	}

	file, err := os.Open(archive.FullPath)

	if err != nil {
		return nil, err
	}

	defer file.Close()

	var chunks []models.Chunk
	chunkSize := getChunkSize(archive.FileSize)

	for {
		data := make([]byte, chunkSize)

		n, err := file.Read(data)

		if err != nil {
			if err != io.EOF {
				utils.Error.Println(err)
			}

			break
		}

		if uint64(n) < chunkSize {
			data = data[:n]
		}

		hash := sha1.Sum(data)
		name := hex.EncodeToString(hash[:])

		if !chunkExists(name, archive) {
			ciphertext := utils.EncryptData(data, getPassword())

			err = saveFile(archive.OutputDir, name+EncSuffix, ciphertext)

			if err != nil {
				return nil, err
			}
		}

		chunks = append(chunks, models.Chunk{
			Name: name,
			Size: uint64(n),
		})
	}

	return chunks, nil
}

func fileHasChanged(archive *ArchiveInfo) bool {
	if archive.File.Size != archive.FileSize {
		return true
	}

	diff := archive.File.ModificationTime.Time.Sub(archive.FileInfo.ModTime())

	if diff < 0 {
		diff *= -1
	}

	if diff > 1*time.Microsecond {
		return true
	}

	return false
}

func getChunkSize(size uint64) uint64 {
	return 1024 * 1024
}

func getPassword() string {
	return *password
}

func markRemovedPaths(removedPaths map[string]bool, doc *models.Document) {
	archive := ArchiveInfo{
		Document: doc,
	}

	for shortPath := range removedPaths {
		utils.Trace.Printf("%s was deleted", shortPath)
		file, exists := doc.Files[shortPath]

		if !exists {
			utils.Error.Printf("markRemovedPaths: %s was in removedPaths, but did not find it in doc.Files", shortPath)
			continue
		}

		archive.File = file
		archive.ShortPath = shortPath

		addToDeletedFiles(&archive)
	}
}

func normalizePath(path string) string {
	path, err := filepath.Abs(path)

	if err != nil {
		utils.Error.Fatalln(err)
	}

	path = filepath.Clean(path)
	path = utils.FixSlashes(path)

	return path
}

func saveFile(outputDir string, filename string, data []byte) error {
	destDir := filepath.Join(outputDir, filename[0:2], filename[0:4])
	destPath := filepath.Join(destDir, filename)

	os.MkdirAll(destDir, 0700)
	err := ioutil.WriteFile(destPath, data, 0700)

	if err != nil {
		return err
	}

	return nil
}

func walkDirectory(inputDir string, outputDir string) {
	utils.Trace.Println("reading index")
	doc, err := readIndex(getDatabaseFilename(outputDir, *plainIndex))

	if err != nil {
		utils.Error.Fatalln(err)
	}

	utils.Trace.Println("creating removed paths map")
	removedPaths := getRemovedPathsMap(doc)

	walkFn := walkDirectoryFn(inputDir, outputDir, doc, removedPaths)

	utils.Trace.Println("checking for changed files")
	filepath.Walk(inputDir, walkFn)

	utils.Trace.Println("checking for deleted files")
	markRemovedPaths(removedPaths, doc)

	utils.Trace.Println("checking for unused chunks")
	chunkIndex := getChunkIndexMap(doc)
	unusedChunks := getUnusedChunks(chunkIndex, outputDir)
	createUnusedChunksDeleteBatch(unusedChunks, outputDir)

	if len(unusedChunks) > 0 {
		utils.Info.Printf("found %d unused chunks", len(unusedChunks))
	}

	utils.Trace.Println("writing to index")
	saveIndex(getDatabaseFilename(outputDir, *plainIndex), doc)
}

func walkDirectoryFn(inputDir string, outputDir string, doc *models.Document, removedPaths removedPathsMap) filepath.WalkFunc {
	inputDirLength := len(inputDir) + 1

	return func(fullPath string, fileInfo os.FileInfo, err error) error {
		fullPath = utils.FixSlashes(fullPath)

		if err != nil {
			utils.Error.Printf("error while walking %s", fullPath)
			return nil
		}

		// Do not walk output directory.
		if fileInfo.IsDir() && strings.HasPrefix(fullPath, outputDir) {
			return filepath.SkipDir
		}

		var shortPath string

		if len(fullPath) >= inputDirLength {
			shortPath = fullPath[inputDirLength:]
		}

		file, exists := doc.Files[shortPath]

		archive := ArchiveInfo{
			Document:  doc,
			File:      file,
			FileInfo:  fileInfo,
			FileSize:  uint64(fileInfo.Size()),
			FullPath:  fullPath,
			InputDir:  inputDir,
			OutputDir: outputDir,
			ShortPath: shortPath,
		}

		if exists {
			delete(removedPaths, shortPath)

			// Fast path for directories.
			if fileInfo.IsDir() {
				file.ModificationTime = models.JSONTime{Time: fileInfo.ModTime()}
				doc.Files[shortPath] = file
				return nil
			}

			if !fileHasChanged(&archive) {
				return nil
			}

			utils.Trace.Printf("%s has changed, updating", shortPath)
			addToDeletedFiles(&archive)

			err := archiveFile(&archive)

			if err != nil {
				utils.Error.Printf("failed archiving %s: %s", shortPath, err)
				return nil
			}
		} else {
			err := archiveFile(&archive)

			if err != nil {
				utils.Error.Printf("failed archiving %s: %s", shortPath, err)
				return nil
			}
		}

		return nil
	}
}
