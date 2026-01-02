package loom

import (
	"bufio"
	"context"
	"io"

	quic "github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
)

type Message struct {
	Key          []byte
	DeclaredSize uint64
	MsgID        uint64
	Reader       io.Reader // streaming reader for body
}

type Consumer struct {
	opt ClientOptions

	q  *quicConn
	h3 *h3Consumer

	br       *bufio.Reader
	ackW     *bufio.Writer
	maxKey   int
	maxChunk int
}

func NewConsumer(ctx context.Context, opt ClientOptions) (*Consumer, error) {
	if opt.Transport == "" {
		opt.Transport = TransportQUIC
	}
	if opt.Name == "" {
		opt.Name = "consumer"
	}
	if opt.Room == "" {
		opt.Room = "default"
	}

	c := &Consumer{opt: opt, maxKey: 256, maxChunk: 64 << 10}
	switch opt.Transport {
	case TransportH3:
		hc, err := newH3Consumer(ctx, opt)
		if err != nil {
			return nil, err
		}
		c.h3 = hc
		c.br = bufio.NewReader(hc.resp.Body)
		c.ackW = hc.bw
	default:
		qc, err := dialQUIC(ctx, opt, "loom")
		if err != nil {
			return nil, err
		}
		c.q = qc
		c.br = qc.r
		c.ackW = qc.w
		if err := WriteHello(qc.w, RoleConsumer, opt.Name, opt.Room, opt.Token); err != nil {
			_ = qc.Close()
			return nil, err
		}
	}
	_ = quic.Version1
	_ = http3.NextProtoH3
	return c, nil
}

func (c *Consumer) Close() error {
	if c.h3 != nil {
		return c.h3.Close()
	}
	if c.q != nil {
		return c.q.Close()
	}
	return nil
}

// Next reads the next message header and returns a streaming reader for its body.
// You must drain msg.Reader to EOF before ACKing, otherwise the stream framing will desync.
func (c *Consumer) Next() (Message, error) {
	h, err := ReadMessageHeader(c.br, c.maxKey)
	if err != nil {
		return Message{}, err
	}
	mr := &messageReader{br: c.br, maxChunk: c.maxChunk}
	return Message{Key: h.Key, DeclaredSize: h.DeclaredSize, MsgID: h.MsgID, Reader: mr}, nil
}

// Consume calls handler for each message, drains to EOM, then ACKs.
func (c *Consumer) Consume(ctx context.Context, handler func(Message) error) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		msg, err := c.Next()
		if err != nil {
			return err
		}
		if err := handler(msg); err != nil {
			// keep framing aligned
			_, _ = io.Copy(io.Discard, msg.Reader)
			return err
		}
		_, _ = io.Copy(io.Discard, msg.Reader)
		if err := WriteAck(c.ackW, msg.MsgID); err != nil {
			return err
		}
	}
}

// DrainAndAck drains msg.Reader to dst and then ACKs msg.MsgID.
func (c *Consumer) DrainAndAck(msg Message, dst io.Writer) error {
	if _, err := io.Copy(dst, msg.Reader); err != nil {
		return err
	}
	return WriteAck(c.ackW, msg.MsgID)
}

type messageReader struct {
	br       *bufio.Reader
	maxChunk int
	done     bool

	buf []byte
	off int
}

func (m *messageReader) Read(p []byte) (int, error) {
	if m.done {
		return 0, io.EOF
	}
	if m.buf != nil && m.off < len(m.buf) {
		n := copy(p, m.buf[m.off:])
		m.off += n
		if m.off >= len(m.buf) {
			m.buf = nil
			m.off = 0
		}
		return n, nil
	}

	// read next chunk without allocating each time
	n, err := readUvarint(m.br)
	if err != nil {
		return 0, err
	}
	if n == 0 {
		m.done = true
		return 0, io.EOF
	}
	if n > uint64(m.maxChunk) {
		return 0, io.ErrUnexpectedEOF
	}
	need := int(n)
	if cap(m.buf) < need {
		m.buf = make([]byte, need)
	} else {
		m.buf = m.buf[:need]
	}
	if _, err := io.ReadFull(m.br, m.buf); err != nil {
		return 0, err
	}

	if len(m.buf) <= len(p) {
		copy(p, m.buf)
		m.buf = nil
		m.off = 0
		return need, nil
	}
	m.off = copy(p, m.buf)
	return m.off, nil
}
