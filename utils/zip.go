package utils

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
)

// CompressData compresses bytes with the gzip algorithm.
func CompressData(data []byte) []byte {
	var output bytes.Buffer

	writer, err := gzip.NewWriterLevel(&output, gzip.BestCompression)

	if err != nil {
		Error.Panicln(err)
	}

	writer.Write(data)
	writer.Close()

	return output.Bytes()
}

// UncompressData uncompresses bytes with the gzip algorithm.
func UncompressData(data []byte) []byte {
	inputRaw := bytes.NewBuffer(data)
	inputZip, err := gzip.NewReader(inputRaw)

	if err != nil {
		Error.Panicln(err)
	}

	output, err := ioutil.ReadAll(inputZip)

	if err != nil {
		Error.Panicln(err)
	}

	inputZip.Close()
	return output
}
