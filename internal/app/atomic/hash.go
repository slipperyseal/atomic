package atomic

import (
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
)

type hashWriter struct {
	hash   hash.Hash
	writer io.Writer
}

func newHashWriter(writer io.Writer) hashWriter {
	h := hashWriter{
		hash:   sha256.New(),
		writer: writer,
	}
	return h
}

func (h hashWriter) Write(p []byte) (n int, err error) {
	h.hash.Write(p)
	return h.writer.Write(p)
}

func (h hashWriter) result() string {
	return fmt.Sprintf("%x", h.hash.Sum(nil))
}
