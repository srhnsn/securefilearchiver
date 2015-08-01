package main

import (
	"crypto/sha1"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/srhnsn/securefilearchiver/models"
	"github.com/srhnsn/securefilearchiver/utils"
)

const (
	indexSaveInterval      = 2 * time.Minute
	progressUpdateInterval = 5 * time.Second
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

// ProgressInfo holds all information that describes the current status
// when archiving files.
type ProgressInfo struct {
	CurrentFile    string
	ProcessedData  uint64
	ProcessedFiles uint64
	SkippedData    uint64
	SkippedFiles   uint64
}

func addToDeletedFiles(archive *ArchiveInfo) {
	archive.File.DeletedAt = &models.JSONTime{Time: time.Now()}
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

func archiveDirectory(archive *ArchiveInfo) {
	archive.File.Chunks = []models.Chunk{}
	archive.File.IsDirectory = true
	archive.File.ModificationTime = models.JSONTime{Time: archive.FileInfo.ModTime()}

	archive.Document.Files[archive.ShortPath] = archive.File
}

func archiveFile(archive *ArchiveInfo) error {
	if archive.FileInfo.IsDir() {
		utils.Error.Panicln("directory passed to archiveFile")
	}

	chunks, err := createAndGetChunks(archive)

	if err != nil {
		return err
	}

	file := models.File{
		Size:             archive.FileSize,
		ModificationTime: models.JSONTime{Time: archive.FileInfo.ModTime()},
		Chunks:           chunks,
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
			ciphertext := utils.EncryptData(data, archive.Document.KeyUnencrypted)

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
		utils.Error.Panicln(err)
	}

	path = filepath.Clean(path)
	path = utils.FixSlashes(path)

	return path
}

func saveFile(outputDir string, filename string, data []byte) error {
	destDir := filepath.Join(outputDir, filename[0:2], filename[0:4])
	destPath := filepath.Join(destDir, filename)

	tmpPath := destPath + TmpSuffix

	os.MkdirAll(destDir, 0700)
	utils.MustWriteFile(tmpPath, data)

	err := os.Rename(tmpPath, destPath)

	if err != nil {
		return err
	}

	return nil
}

func walkDirectory(inputDir string, outputDir string) {
	doc, err := readIndex(getExistingIndexFilename(outputDir))

	if err != nil {
		utils.Error.Panicln(err)
	}

	utils.Trace.Println("creating removed paths map")
	removedPaths := getRemovedPathsMap(doc)

	var progressInfo ProgressInfo
	done := make(chan bool)
	saveTicker := time.NewTicker(indexSaveInterval)
	startProgressUpdater(&progressInfo, done)
	walkFn := walkDirectoryFn(inputDir, outputDir, doc, removedPaths, &progressInfo, saveTicker.C)

	utils.Trace.Println("checking for changed files")
	filepath.Walk(inputDir, walkFn)
	done <- true
	saveTicker.Stop()

	utils.Trace.Println("checking for deleted files")
	markRemovedPaths(removedPaths, doc)

	utils.Trace.Println("writing to index")
	saveIndex(getIndexFilename(outputDir), doc)
}

func walkDirectoryFn(
	inputDir string,
	outputDir string,
	doc *models.Document,
	removedPaths removedPathsMap,
	progressInfo *ProgressInfo,
	save <-chan time.Time,
) filepath.WalkFunc {

	inputDirLength := len(inputDir) + 1

	var excludes utils.Globfile
	var err error

	if len(*archiveExcludes) != 0 {
		excludes, err = utils.NewGlobfile(*archiveExcludes)

		if err != nil {
			utils.Error.Panicln(err)
		}

		utils.Trace.Printf("using exclude file %s (%d globs)", *archiveExcludes, excludes.Len())
	}

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

		if fileInfo.Mode()&os.ModeSymlink == os.ModeSymlink && !*archiveSymlinks {
			return filepath.SkipDir
		}

		var shortPath string

		if len(fullPath) >= inputDirLength {
			shortPath = fullPath[inputDirLength:]
		}

		if excludes.Matches(shortPath) {
			utils.Trace.Printf("skipping %s because path is in exclude file", shortPath)

			if fileInfo.IsDir() {
				return filepath.SkipDir
			}

			return nil
		}

		progressInfo.CurrentFile = shortPath
		delete(removedPaths, shortPath)

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

		// Fast path for directories as they do not need chunks.
		if fileInfo.IsDir() {
			archiveDirectory(&archive)
			return nil
		}

		if exists {
			if fileInfo.IsDir() {
				file.ModificationTime = models.JSONTime{Time: fileInfo.ModTime()}
				doc.Files[shortPath] = file
				return nil
			}

			if !fileHasChanged(&archive) {
				progressInfo.SkippedData += file.Size
				progressInfo.SkippedFiles++
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

		progressInfo.ProcessedFiles++
		progressInfo.ProcessedData += archive.FileSize

		select {
		case <-save:
			utils.Info.Println("doing intermediary index save")
			saveIndex(getIndexFilename(outputDir), doc)
			utils.Info.Println("continuing archive process")
		default:
		}

		return nil
	}
}

func printProgress(
	progressInfo *ProgressInfo,
	lastTickDuration time.Duration,
	lastTickData uint64,
	totalDuration time.Duration,
) {

	processedDataFormatted := utils.FormatFileSize(progressInfo.ProcessedData)
	skippedDataFormatted := utils.FormatFileSize(progressInfo.SkippedData)

	speed := lastTickData / uint64(lastTickDuration.Seconds())
	speedFormatted := utils.FormatFileSize(speed) + "/s"

	utils.Info.Printf(`Archive process started %s ago.
Processed files: %d
Processed data:  %s
Skipped files:   %d
Skipped data:    %s
Current file:    %s
Current speed:   %s

`,
		totalDuration,
		progressInfo.ProcessedFiles,
		processedDataFormatted,
		progressInfo.SkippedFiles,
		skippedDataFormatted,
		progressInfo.CurrentFile,
		speedFormatted,
	)
}

func startProgressUpdater(progressInfo *ProgressInfo, done chan bool) {
	go func() {
		lastTotalProcessedData := progressInfo.ProcessedData
		lastTick := time.Now()
		start := time.Now()
		ticker := time.NewTicker(progressUpdateInterval)

		for {
			select {
			case <-ticker.C:
				now := time.Now()

				lastTickDuration := now.Sub(lastTick)
				lastTick = now

				lastTickData := progressInfo.ProcessedData - lastTotalProcessedData
				lastTotalProcessedData = progressInfo.ProcessedData

				totalDuration := now.Sub(start)
				totalDuration = time.Duration(totalDuration.Seconds()) * time.Second

				printProgress(
					progressInfo,
					lastTickDuration,
					lastTickData,
					totalDuration,
				)

			case <-done:
				ticker.Stop()
				return
			}
		}
	}()
}
