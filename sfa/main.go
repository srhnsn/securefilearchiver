package main

import (
	"os"

	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/srhnsn/securefilearchiver/utils"
)

const (
	// EncSuffix is the suffix for all encrypted (binary) files.
	EncSuffix = ".bin"
	// ZipSuffix is the suffix for all compressed files.
	ZipSuffix = ".gz"
)

var (
	app        = kingpin.New("sfa", "A secure file archiver.")
	verbose    = app.Flag("verbose", "Verbose output.").Short('v').Bool()
	noIndexEnc = app.Flag("noindexenc", "Do not encrypt index file.").Bool()
	noIndexZip = app.Flag("noindexzip", "Do not compress index file.").Bool()
	password   = app.Flag("password", "Password to use for encryption and decryption.").String()

	archive          = app.Command("archive", "Archive files.")
	archiveInputDir  = archive.Arg("source", "Source directory.").Required().String()
	archiveOutputDir = archive.Arg("destination", "Destination directory").Required().String()
	archiveExcludes  = archive.Flag("exclude-file", "Never archive paths that match the globs in this file.").String()
	archiveSymlinks  = archive.Flag("follow-symlinks", "Follow and archive symbolic links. They are ignored otherwise.").Bool()

	restore          = app.Command("restore", "Restore files.")
	restoreInputDir  = restore.Arg("source", "Source directory.").Required().String()
	restoreOutputDir = restore.Arg("destination", "Destination directory.").Required().String()
	restorePattern   = restore.Flag("pattern", "A glob pattern to selectively restore files.").String()

	indexCmd      = app.Command("index", "Index operations.")
	indexInputDir = indexCmd.Arg("source", "Source directory.").Required().String()
	indexPrune    = indexCmd.Flag("prune", "Prune deleted files older than a specific time range.").String()
	indexGC       = indexCmd.Flag("gc", "Remove unused chunks.").Bool()
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

	case indexCmd.FullCommand():
		input := normalizePath(*indexInputDir)

		if len(*indexPrune) != 0 {
			pruneFiles(input, *indexPrune)
		}

		if *indexGC {
			garbageCollect(input)
		}
	}
}
