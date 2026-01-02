package loom

import (
	"bufio"
	"encoding/binary"
	"io"
)

func writeUvarint(w io.Writer, v uint64) error {
	var buf [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(buf[:], v)
	_, err := w.Write(buf[:n])
	return err
}

func readUvarint(r *bufio.Reader) (uint64, error) {
	return binary.ReadUvarint(r)
}
