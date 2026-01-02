package loom

import (
	"bufio"
	"bytes"
	"io"
	"testing"
)

func TestProtocolRoundTrip(t *testing.T) {
	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	if err := WriteHello(w, RoleProducer, "p", "r", "t"); err != nil {
		t.Fatal(err)
	}
	if err := WriteMessageHeader(w, []byte("k"), 9, 7); err != nil {
		t.Fatal(err)
	}
	if err := WriteChunk(w, []byte("abc")); err != nil {
		t.Fatal(err)
	}
	if err := WriteEndOfMessage(w); err != nil {
		t.Fatal(err)
	}
	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}

	r := bufio.NewReader(bytes.NewReader(b.Bytes()))
	role, name, room, token, err := ReadHello(r, 8, 8, 8)
	if err != nil {
		t.Fatal(err)
	}
	if role != RoleProducer || name != "p" || room != "r" || token != "t" {
		t.Fatalf("bad hello")
	}
	h, err := ReadMessageHeader(r, 8)
	if err != nil {
		t.Fatal(err)
	}
	if string(h.Key) != "k" || h.DeclaredSize != 9 || h.MsgID != 7 {
		t.Fatalf("bad header: %+v", h)
	}
	chunk, done, err := ReadChunk(r, 32)
	if err != nil {
		t.Fatal(err)
	}
	if done || string(chunk) != "abc" {
		t.Fatalf("bad chunk")
	}
	_, done, err = ReadChunk(r, 32)
	if err != nil {
		t.Fatal(err)
	}
	if !done {
		t.Fatal("expected eom")
	}
}

func TestMessageReaderSupportsSmallReads(t *testing.T) {
	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	if err := WriteChunk(w, []byte("hello")); err != nil {
		t.Fatal(err)
	}
	if err := WriteChunk(w, []byte("world")); err != nil {
		t.Fatal(err)
	}
	if err := WriteEndOfMessage(w); err != nil {
		t.Fatal(err)
	}
	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}

	mr := &messageReader{br: bufio.NewReader(bytes.NewReader(b.Bytes())), maxChunk: 64 << 10}
	out, err := io.ReadAll(mr)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "helloworld" {
		t.Fatalf("got %q", string(out))
	}
}
