package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ryanuber/go-glob"

	"github.com/srhnsn/securefilearchiver/models"
	"github.com/srhnsn/securefilearchiver/utils"
)

const (
	outputScriptfile = "restore.bat"
	mtimeFormat      = "2006-01-02 15:04:05.999999999 -0700"
	passwordFile     = "key.txt"
)

func getRestoreDirectoryCommands(inputDir string, outputDir string, shortPath string, file models.File) []string {
	out := []string{}

	out = append(out, getMkDirCmd(shortPath))
	out = append(out, getTouchCmd(shortPath, file.ModificationTime.Time))

	return out
}

func getRestoreFileCommands(inputDir string, outputDir string, shortPath string, file models.File) []string {
	out := []string{}

	destDir := filepath.Dir(shortPath)
	filename := filepath.Base(shortPath)

	out = append(out, getMkDirCmd(destDir))
	var chunkCmds []string

	if len(file.Chunks) == 1 {
		chunkCmds = restoreSingleChunk(inputDir, destDir, filename, file)
	} else {
		chunkCmds = restoreMultipleChunks(inputDir, destDir, filename, file)
	}

	out = append(out, chunkCmds...)
	out = append(out, getTouchCmd(filepath.Join(destDir, filename), file.ModificationTime.Time))

	return out
}

func getRestorePathsCommands(inputDir string, outputDir string, doc *models.Document) ([]string, uint64) {
	out := []string{}

	var noFiles uint64

	sortedKeys := doc.GetSortedFilesKeys()

	for _, shortPath := range sortedKeys {
		file := doc.Files[shortPath]

		if len(*restorePattern) != 0 && len(shortPath) != 0 && !glob.Glob(*restorePattern, shortPath) {
			continue
		}

		noFiles++
		var cmds []string

		if len(shortPath) == 0 {
			shortPath = "."
		}

		if file.IsDirectory {
			cmds = getRestoreDirectoryCommands(inputDir, outputDir, shortPath, file)
		} else {
			cmds = getRestoreFileCommands(inputDir, outputDir, shortPath, file)
		}

		out = append(out, cmds...)
		out = append(out, "")
	}

	return out, noFiles
}

func restoreFiles(inputDir string, outputDir string) {
	doc, err := readIndex(getExistingIndexFilename(inputDir))

	utils.PanicIfErr(err)

	if len(*restorePattern) != 0 {
		utils.Info.Printf("using restore pattern %s", *restorePattern)
	}

	out := []string{
		"@echo off",
		"",
		"chcp 65001 >NUL",
		"",
	}

	restoreCommands, noFiles := getRestorePathsCommands(inputDir, outputDir, doc)
	out = append(out, restoreCommands...)

	if len(*restorePattern) == 0 {
		utils.Info.Printf("restored %d files", len(doc.Files))
	} else {
		utils.Info.Printf("restored %d out of %d files", noFiles, len(doc.Files))
	}

	out = append(out, "pause")
	err = os.MkdirAll(outputDir, 0700)

	utils.PanicIfErr(err)

	data := []byte(strings.Join(out, "\r\n"))
	utils.MustWriteFile(filepath.Join(outputDir, outputScriptfile), data)

	utils.MustWriteFile(filepath.Join(outputDir, passwordFile), []byte(doc.KeyUnencrypted))
}

func restoreSingleChunk(inputDir string, destDir string, filename string, file models.File) []string {
	out := []string{}

	chunk := file.Chunks[0]
	chunkSource := filepath.Join(inputDir, chunk.Name[0:2], chunk.Name[0:4], chunk.Name+EncSuffix)
	chunkDest := filepath.Join(destDir, filename)

	cmd := utils.GetDecryptCommand(chunkSource, chunkDest, passwordFile)
	out = append(out, cmd)

	return out
}

func restoreMultipleChunks(inputDir string, destDir string, filename string, file models.File) []string {
	out := []string{}

	fileDest := filepath.Join(destDir, filename)
	concatList := []string{}
	delList := []string{}

	for chunkNo, chunk := range file.Chunks {
		chunkSource := filepath.Join(inputDir, chunk.Name[0:2], chunk.Name[0:4], chunk.Name+EncSuffix)
		chunkDest := filepath.Join(destDir, fmt.Sprintf("%s.%d", filename, chunkNo+1))

		cmd := utils.GetDecryptCommand(chunkSource, chunkDest, passwordFile)

		out = append(out, cmd)
		concatList = append(concatList, chunkDest)
		delList = append(delList, getDeleteCmd(chunkDest))
	}

	out = append(out, getConcatCmd(concatList, fileDest))
	out = append(out, delList...)

	return out
}

func getConcatCmd(files []string, dest string) string {
	return fmt.Sprintf(
		`copy /B /Y "%s" "%s" >NUL`,
		strings.Join(files, `"+"`),
		dest,
	)
}

func getDeleteCmd(path string) string {
	return fmt.Sprintf(`del "%s"`, path)
}

func getMkDirCmd(dir string) string {
	return fmt.Sprintf(`mkdir "%s" >NUL 2>&1`, dir)
}

func getTouchCmd(path string, mtime time.Time) string {
	return fmt.Sprintf(
		`call touch -d "%s" "%s"`,
		mtime.Format(mtimeFormat),
		path,
	)
}
