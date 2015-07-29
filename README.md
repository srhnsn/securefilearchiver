# Secure File Archiver

Secure File Archiver (SFA) is a tool for securely storing files in locations
you do not trust (i. e. Dropbox and the like).

# Installation

`go get github.com/srhnsn/securefilearchiver/...`

## Usage

    usage: sfa [<flags>] <command> [<args> ...]

    A secure file archiver.

    Flags:
      --help         Show help (also see --help-long and --help-man).
      -v, --verbose  Verbose output.
      --noindexenc   Do not encrypt index file.
      --noindexzip   Do not compress index file.
      --pass=PASS    Pass phrase argument that is passed as-is to OpenSSL's -pass.

    Commands:
      help [<command>...]
        Show help.


      archive [<flags>] <source> <destination>
        Archive files.

        --follow-symlinks  Follow and archive symbolic links. They are ignored
                           otherwise.

      restore [<flags>] <source> <destination>
        Restore files.

        --pattern=PATTERN  A glob pattern to selectively restore files.

# Technical overview

## Archiving files

1. First, you specify a directory for SFA to scan.
1. SFA splits the files in the directory in chunks with a specific maximum size for better
   cloud software compatibility and stores them in a directory.  
   (Try uploading an encrypted 250 GB container file to Dropbox.)
1. Each chunk is encrypted by using symmetric OpenSSL encryption. The used command is:  
   `openssl enc -aes-256-cbc -e -S <salt> -pass <password>`  
   `Stdin` and `stdout` are used for data transfer.
1. An encrypted `index.json.bin` is generated which stores this information about each file:
    1. Filename
    1. Modification date
    1. Size
    1. Associated chunks:
        1. Filename (which is the SHA-1 checksum of the chunk content)  
           The SHA-1 checksums are also used for deduplication.
        1. Chunk order

Your files are encrypted with a generated 256 bit key. This key is encrypted with your own
key and stored in the index file. The index file is, again, encrypted with your key.

## Restoring files

SFA does not do any restoration itself. Instead, it generates batch files which only use
standard Windows/Unix software. The reason for this is, I (and maybe you, too) want to
be independant from third-party backup software as much as possible, in case I need to
inspect or repair the backed up files some day.

After the `index.json.bin` is read, these are the commands that are used to restore
each file:

1. For each chunk of each file a specific OpenSSL command is generated:  
   `openssl enc -aes-256-cbc -d -pass <password> -in <encrypted_chunk> -out <decrypted_chunk>`
1. On Windows, decrypted chunks are concatenated with the `copy` command:  
   `copy /B <chunk_1>+<chunk_2>+...+<chunk_n> <original_filename>`
1. Modification times are restored with `touch`.

# Security considerations

1. File content should be secure.
1. Filenames should be secure.
1. File sizes are not necessarily visible if
    1. the file size is greater than the chunk size and
    1. multiple files where 1. applies are updated (or else the remote host can check
       which chunks are updated together).
1. Exact number of files should not be visible, however, there is some undeniable correlation
   between the number of files and the number of associated chunks.
1. If the `index.json.bin` is lost, the archive will be pretty much useless.

# Questions and answers

### Why not use [Obnam](http://obnam.org/)?

Obnam looks great! I first read about it in
[this article](http://changelog.complete.org/archives/9353-roundup-of-remote-encrypted-deduplicated-backups-in-linux).
However ... I'm a Windows user! This is not going to change any time soon, for several
reasons, and I still need a reliable backup solution.

### Can I actually use this?

This is yet another Work In Progress™ and I am not using it for my own backups so far, so
neither should you.

# To do

1. Index status viewer.
1. Old snapshots should be pruned at some point.
1. Tests.
1. A way to determine good chunk sizes should be found. Current threshold is fixed at 1 MiB.
1. Actual Linux support.
