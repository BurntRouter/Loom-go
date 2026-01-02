package loom

import (
	"bufio"
	"context"
	"io"

	"github.com/quic-go/quic-go/http3"
)

type Producer struct {
	opt ClientOptions

	// one of:
	q  *quicConn
	h3 *h3Producer

	bw *bufio.Writer
}

func NewProducer(ctx context.Context, opt ClientOptions) (*Producer, error) {
	if opt.Transport == "" {
		opt.Transport = TransportQUIC
	}
	if opt.Name == "" {
		opt.Name = "producer"
	}
	if opt.Room == "" {
		opt.Room = "default"
	}

	p := &Producer{opt: opt}
	switch opt.Transport {
	case TransportH3:
		hp, err := newH3Producer(ctx, opt)
		if err != nil {
			return nil, err
		}
		p.h3 = hp
		p.bw = hp.bw
	default:
		qc, err := dialQUIC(ctx, opt, "loom")
		if err != nil {
			return nil, err
		}
		p.q = qc
		p.bw = qc.w
		if err := WriteHello(p.bw, RoleProducer, opt.Name, opt.Room, opt.Token); err != nil {
			_ = qc.Close()
			return nil, err
		}
	}
	_ = http3.NextProtoH3 // keep dependency tidy in case only QUIC is used
	return p, nil
}

func (p *Producer) Close() error {
	if p.h3 != nil {
		return p.h3.Close()
	}
	if p.q != nil {
		return p.q.Close()
	}
	return nil
}

// Produce streams a single message. msgID is ignored by server; pass 0.
func (p *Producer) Produce(ctx context.Context, key []byte, declaredSize uint64, msgID uint64, r io.Reader, chunkSize int) error {
	_ = ctx
	if chunkSize <= 0 {
		chunkSize = 64 << 10
	}
	if err := WriteMessageHeader(p.bw, key, declaredSize, msgID); err != nil {
		return err
	}
	buf := make([]byte, chunkSize)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			if err := WriteChunk(p.bw, buf[:n]); err != nil {
				return err
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
	}
	if err := WriteEndOfMessage(p.bw); err != nil {
		return err
	}
	if err := p.bw.Flush(); err != nil {
		return err
	}
	return nil
}

// Close() waits for the HTTP/3 response; this is a no-op for compatibility.
func (p *Producer) FinalizeHTTP3() error { return nil }
