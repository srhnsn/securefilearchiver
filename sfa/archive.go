package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/srhnsn/securefilearchiver/models"
	"github.com/srhnsn/securefilearchiver/utils"
)

const (
	indexSaveInterval      = 5 * time.Minute
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

func archiveFile(archive *ArchiveInfo, exists bool) error {
	file := models.File{
		ModificationTime: models.JSONTime{Time: archive.FileInfo.ModTime()},
	}

	if archive.FileInfo.IsDir() {
		file.IsDirectory = true
	} else {
		chunks, err := createAndGetChunks(archive)

		if err != nil {
			return err
		}

		file.Chunks = chunks
		file.Size = archive.FileSize
	}

	if exists {
		file.AddedAt = archive.File.AddedAt
	} else {
		file.AddedAt = models.JSONTime{Time: time.Now()}
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

	chunkNo := 0

	utils.Trace.Printf("writing chunks for %s\n", archive.ShortPath)

	for {
		chunkNo++
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

		name := utils.GetHashSum(data)
		chunkFilename := name + EncSuffix

		if chunkExists(name, archive) {
			utils.Trace.Printf("chunk #%d (%s) seems to already exist\n", chunkNo, chunkFilename)
		} else {
			utils.Trace.Printf("writing chunk #%d (%s)\n", chunkNo, chunkFilename)
			ciphertext := utils.EncryptData(data, archive.Document.KeyUnencrypted)

			saveChunk(archive.OutputDir, chunkFilename, ciphertext)
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

	utils.PanicIfErr(err)

	path = filepath.Clean(path)
	path = utils.FixSlashes(path)

	return path
}

func saveChunk(outputDir string, filename string, data []byte) {
	destDir := filepath.Join(outputDir, filename[0:2], filename[0:4])
	destPath := filepath.Join(destDir, filename)

	err := os.MkdirAll(destDir, 0700)

	utils.PanicIfErr(err)

	utils.MustWriteFileAtomic(destPath, data)
}

func walkDirectory(inputDir string, outputDir string) {
	doc, err := readIndex(getExistingIndexFilename(outputDir))

	utils.PanicIfErr(err)

	utils.Trace.Println("creating removed paths map")
	removedPaths := getRemovedPathsMap(doc)

	var progressInfo ProgressInfo
	done := make(chan bool)
	saveTicker := time.NewTicker(indexSaveInterval)
	startProgressUpdater(&progressInfo, done)
	walkFn := walkDirectoryFn(inputDir, outputDir, doc, removedPaths, &progressInfo, saveTicker.C)

	utils.Info.Println("checking for changed files")
	err = filepath.Walk(inputDir, walkFn)

	utils.PanicIfErr(err)

	done <- true
	saveTicker.Stop()

	utils.Info.Println("checking for deleted files")
	markRemovedPaths(removedPaths, doc)

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

		utils.PanicIfErr(err)

		utils.Info.Printf("using exclude file %s (%d globs)", *archiveExcludes, excludes.Len())
	}

	return func(fullPath string, fileInfo os.FileInfo, err error) error {
		fullPath = utils.FixSlashes(fullPath)

		if err != nil {
			utils.Error.Printf("error while walking %s: %s", fullPath, err)
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

		// Fast path for directories as they do not need chunks and snapshots.
		if fileInfo.IsDir() {
			err := archiveFile(&archive, exists)
			utils.PanicIfErr(err)
			return nil
		}

		if exists {
			if !fileHasChanged(&archive) {
				utils.Trace.Printf("skipping unchanged file %s", shortPath)
				progressInfo.SkippedData += file.Size
				progressInfo.SkippedFiles++
				return nil
			}

			utils.Trace.Printf("updating changed file %s", shortPath)
			addToDeletedFiles(&archive)
		} else {
			utils.Trace.Printf("adding new file %s", shortPath)
		}

		err = archiveFile(&archive, exists)

		if err != nil {
			utils.Error.Println(err)
			return nil
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
	lastTickFiles uint64,
	totalDuration time.Duration,
) {

	processedDataFormatted := utils.FormatFileSize(progressInfo.ProcessedData)
	skippedDataFormatted := utils.FormatFileSize(progressInfo.SkippedData)

	transferRate := lastTickData / uint64(lastTickDuration.Seconds())
	transferRateFormatted := utils.FormatFileSize(transferRate) + "/s"

	fileRate := lastTickFiles / uint64(lastTickDuration.Seconds())

	utils.Info.Printf(`Archive process started %s ago.
Processed files: %d
Processed data:  %s
Skipped files:   %d
Skipped data:    %s
Current file:    %s
Transfer rate:   %s
File rate:       %d files/s

`,
		totalDuration,
		progressInfo.ProcessedFiles,
		processedDataFormatted,
		progressInfo.SkippedFiles,
		skippedDataFormatted,
		progressInfo.CurrentFile,
		transferRateFormatted,
		fileRate,
	)
}

func startProgressUpdater(progressInfo *ProgressInfo, done chan bool) {
	go func() {
		lastTotalProcessedData := progressInfo.ProcessedData
		lastTotalProcessedFiles := progressInfo.ProcessedFiles
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

				lastTickFiles := progressInfo.ProcessedFiles - lastTotalProcessedFiles
				lastTotalProcessedFiles = progressInfo.ProcessedFiles

				totalDuration := now.Sub(start)
				totalDuration = time.Duration(totalDuration.Seconds()) * time.Second

				printProgress(
					progressInfo,
					lastTickDuration,
					lastTickData,
					lastTickFiles,
					totalDuration,
				)

			case <-done:
				ticker.Stop()
				return
			}
		}
	}()
}
