package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ryanuber/go-glob"

	"github.com/srhnsn/securefilearchiver/models"
	"github.com/srhnsn/securefilearchiver/utils"
)

const outputScriptfile = "restore.bat"
const mtimeFormat = "2006-01-02 15:04:05.999999999 -0700"

func getRestoreFileCommands(inputDir string, outputDir string, shortPath string, file models.File) []string {
	out := []string{}

	destDir := filepath.Join(outputDir, filepath.Dir(shortPath))
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

func getRestoreFilesCommands(inputDir string, outputDir string, doc *models.Document) ([]string, uint64) {
	out := []string{}

	var noFiles uint64

	for shortPath, file := range doc.Files {
		if len(*restorePattern) != 0 && !glob.Glob(*restorePattern, shortPath) {
			continue
		}

		noFiles += 1
		cmds := getRestoreFileCommands(inputDir, outputDir, shortPath, file)
		out = append(out, cmds...)
		out = append(out, "")
	}

	return out, noFiles
}

func restoreFiles(inputDir string, outputDir string) {
	utils.Trace.Println("reading index")
	doc, err := readIndex(getDatabaseFilename(inputDir, *plainIndex))

	if err != nil {
		utils.Error.Fatalln(err)
	}

	if len(*restorePattern) != 0 {
		utils.Info.Printf("using restore pattern %s", *restorePattern)
	}

	out := []string{"@echo off", ""}

	restoreCommands, noFiles := getRestoreFilesCommands(inputDir, outputDir, doc)
	out = append(out, restoreCommands...)

	if len(*restorePattern) == 0 {
		utils.Info.Printf("restored %d files", len(doc.Files))
	} else {
		utils.Info.Printf("restored %d out of %d files", noFiles, len(doc.Files))
	}

	out = append(out, "pause")
	err = os.MkdirAll(outputDir, 0700)

	if err != nil {
		utils.Error.Fatalln(err)
	}

	data := []byte(strings.Join(out, "\r\n"))
	err = ioutil.WriteFile(filepath.Join(outputDir, outputScriptfile), data, 0700)

	if err != nil {
		utils.Error.Fatalln(err)
	}
}

func restoreSingleChunk(inputDir string, destDir string, filename string, file models.File) []string {
	out := []string{}

	chunk := file.Chunks[0]
	chunkSource := filepath.Join(inputDir, chunk.Name[0:2], chunk.Name[0:4], chunk.Name+EncSuffix)
	chunkDest := filepath.Join(destDir, filename)

	cmd := fmt.Sprintf(
		`openssl enc -aes-256-cbc -d -pass "%s" -in "%s" -out "%s"`,
		getPassword(),
		chunkSource,
		chunkDest,
	)

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

		cmd := fmt.Sprintf(
			`openssl enc -aes-256-cbc -d -pass "%s" -in "%s" -out "%s"`,
			getPassword(),
			chunkSource,
			chunkDest,
		)

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
		`copy /B "%s" %s >NUL`,
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
		`touch -d "%s" "%s"`,
		mtime.Format(mtimeFormat),
		path,
	)
}
