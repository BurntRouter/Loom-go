package loom

import (
	"bufio"
	"errors"
	"fmt"
	"io"
)

const (
	Magic       = "LOOM"
	VersionByte = 4

	RoleProducer = byte('P')
	RoleConsumer = byte('C')

	FrameAck = uint64(1)
)

var ErrBadHandshake = errors.New("loom: bad handshake")

func WriteHello(w *bufio.Writer, role byte, name, room, token string) error {
	if role != RoleProducer && role != RoleConsumer {
		return fmt.Errorf("loom: unknown role %q", role)
	}
	if _, err := w.WriteString(Magic); err != nil {
		return err
	}
	if err := w.WriteByte(VersionByte); err != nil {
		return err
	}
	if err := w.WriteByte(role); err != nil {
		return err
	}
	if err := writeUvarint(w, uint64(len(name))); err != nil {
		return err
	}
	if _, err := w.WriteString(name); err != nil {
		return err
	}
	if err := writeUvarint(w, uint64(len(room))); err != nil {
		return err
	}
	if _, err := w.WriteString(room); err != nil {
		return err
	}
	if err := writeUvarint(w, uint64(len(token))); err != nil {
		return err
	}
	if _, err := w.WriteString(token); err != nil {
		return err
	}
	return w.Flush()
}

func ReadHello(r *bufio.Reader, maxName, maxRoom, maxToken int) (role byte, name, room, token string, err error) {
	preface := make([]byte, len(Magic)+2)
	if _, err := io.ReadFull(r, preface); err != nil {
		return 0, "", "", "", err
	}
	if string(preface[:len(Magic)]) != Magic || preface[len(Magic)] != VersionByte {
		return 0, "", "", "", ErrBadHandshake
	}
	role = preface[len(Magic)+1]

	nameLen, err := readUvarint(r)
	if err != nil {
		return 0, "", "", "", err
	}
	if nameLen > uint64(maxName) {
		return 0, "", "", "", fmt.Errorf("loom: name too large: %d", nameLen)
	}
	nameBuf := make([]byte, int(nameLen))
	if _, err := io.ReadFull(r, nameBuf); err != nil {
		return 0, "", "", "", err
	}

	roomLen, err := readUvarint(r)
	if err != nil {
		return 0, "", "", "", err
	}
	if roomLen > uint64(maxRoom) {
		return 0, "", "", "", fmt.Errorf("loom: room too large: %d", roomLen)
	}
	roomBuf := make([]byte, int(roomLen))
	if _, err := io.ReadFull(r, roomBuf); err != nil {
		return 0, "", "", "", err
	}

	tokLen, err := readUvarint(r)
	if err != nil {
		return 0, "", "", "", err
	}
	if tokLen > uint64(maxToken) {
		return 0, "", "", "", fmt.Errorf("loom: token too large: %d", tokLen)
	}
	tokBuf := make([]byte, int(tokLen))
	if _, err := io.ReadFull(r, tokBuf); err != nil {
		return 0, "", "", "", err
	}

	return role, string(nameBuf), string(roomBuf), string(tokBuf), nil
}

type MessageHeader struct {
	Key          []byte
	DeclaredSize uint64
	MsgID        uint64
}

func WriteMessageHeader(w *bufio.Writer, key []byte, declaredSize uint64, msgID uint64) error {
	if len(key) == 0 {
		return errors.New("loom: empty key")
	}
	if err := writeUvarint(w, uint64(len(key))); err != nil {
		return err
	}
	if _, err := w.Write(key); err != nil {
		return err
	}
	if err := writeUvarint(w, declaredSize); err != nil {
		return err
	}
	return writeUvarint(w, msgID)
}

func ReadMessageHeader(r *bufio.Reader, maxKey int) (MessageHeader, error) {
	keyLen, err := readUvarint(r)
	if err != nil {
		return MessageHeader{}, err
	}
	if keyLen == 0 {
		return MessageHeader{}, errors.New("loom: empty key")
	}
	if keyLen > uint64(maxKey) {
		return MessageHeader{}, fmt.Errorf("loom: key too large: %d", keyLen)
	}
	key := make([]byte, int(keyLen))
	if _, err := io.ReadFull(r, key); err != nil {
		return MessageHeader{}, err
	}
	sz, err := readUvarint(r)
	if err != nil {
		return MessageHeader{}, err
	}
	msgID, err := readUvarint(r)
	if err != nil {
		return MessageHeader{}, err
	}
	return MessageHeader{Key: key, DeclaredSize: sz, MsgID: msgID}, nil
}

func WriteChunk(w *bufio.Writer, b []byte) error {
	if err := writeUvarint(w, uint64(len(b))); err != nil {
		return err
	}
	if len(b) == 0 {
		return nil
	}
	_, err := w.Write(b)
	return err
}

func WriteEndOfMessage(w *bufio.Writer) error {
	return writeUvarint(w, 0)
}

func ReadChunk(r *bufio.Reader, maxChunk int) (chunk []byte, done bool, err error) {
	n, err := readUvarint(r)
	if err != nil {
		return nil, false, err
	}
	if n == 0 {
		return nil, true, nil
	}
	if n > uint64(maxChunk) {
		return nil, false, fmt.Errorf("loom: chunk too large: %d", n)
	}
	buf := make([]byte, int(n))
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, false, err
	}
	return buf, false, nil
}

func WriteAck(w *bufio.Writer, msgID uint64) error {
	if err := writeUvarint(w, FrameAck); err != nil {
		return err
	}
	if err := writeUvarint(w, msgID); err != nil {
		return err
	}
	return w.Flush()
}
