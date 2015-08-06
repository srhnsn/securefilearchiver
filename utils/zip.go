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
	PanicIfErr(err)

	_, err = writer.Write(data)
	PanicIfErr(err)

	err = writer.Close()
	PanicIfErr(err)

	return output.Bytes()
}

// UncompressData uncompresses bytes with the gzip algorithm.
func UncompressData(data []byte) []byte {
	inputRaw := bytes.NewBuffer(data)

	inputZip, err := gzip.NewReader(inputRaw)
	PanicIfErr(err)

	output, err := ioutil.ReadAll(inputZip)
	PanicIfErr(err)

	err = inputZip.Close()
	PanicIfErr(err)

	return output
}
