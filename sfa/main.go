package main

import (
	"os"
	"path/filepath"

	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/srhnsn/securefilearchiver/utils"
)

// EncSuffix is the suffix for all encrypted (binary) files.
const EncSuffix = ".bin"

var (
	app        = kingpin.New("sfa", "A secure file archiver.")
	verbose    = app.Flag("verbose", "Verbose output.").Short('v').Bool()
	plainIndex = app.Flag("plainindex", "Do not encrypt index file.").Bool()
	password   = app.Flag("pass", "Pass phrase argument that is passed as-is to OpenSSL's -pass.").String()

	archive          = app.Command("archive", "Archive files.")
	archiveInputDir  = archive.Arg("source", "Source directory.").Required().String()
	archiveOutputDir = archive.Arg("destination", "Destination directory").Required().String()

	restore          = app.Command("restore", "Restore files.")
	restoreInputDir  = restore.Arg("source", "Source directory.").Required().String()
	restoreOutputDir = restore.Arg("destination", "Destination directory.").Required().String()
	restorePattern   = restore.Flag("pattern", "A glob pattern to selectively restore files.").String()
)

func main() {
	cmd := kingpin.MustParse(app.Parse(os.Args[1:]))

	utils.SetVerboseLogging(*verbose)

	switch cmd {
	case archive.FullCommand():
		input := normalizePath(*archiveInputDir)
		output := normalizePath(*archiveOutputDir)

		walkDirectory(input, output)

	case restore.FullCommand():
		input := normalizePath(*restoreInputDir)
		output := normalizePath(*restoreOutputDir)

		restoreFiles(input, output)
	}
}

func getDatabaseFilename(directory string, plainIndex bool) string {
	filename := filepath.Join(directory, databaseFilename)

	if !plainIndex {
		filename += EncSuffix
	}

	return filename
}
