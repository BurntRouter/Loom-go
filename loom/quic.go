package loom

import (
	"bufio"
	"context"
	"errors"

	quic "github.com/quic-go/quic-go"
)

type quicConn struct {
	conn   *quic.Conn
	stream *quic.Stream
	r      *bufio.Reader
	w      *bufio.Writer
}

func dialQUIC(ctx context.Context, opt ClientOptions, nextProto string) (*quicConn, error) {
	tlsCfg, err := tlsConfig(opt.TLS, []string{nextProto})
	if err != nil {
		return nil, err
	}
	c, err := quic.DialAddr(ctx, opt.Addr, tlsCfg, &quic.Config{})
	if err != nil {
		return nil, err
	}
	st, err := c.OpenStreamSync(ctx)
	if err != nil {
		_ = c.CloseWithError(0, "")
		return nil, err
	}
	qc := &quicConn{conn: c, stream: st, r: bufio.NewReader(st), w: bufio.NewWriter(st)}
	return qc, nil
}

func (q *quicConn) Close() error {
	if q == nil {
		return nil
	}
	var err error
	if q.stream != nil {
		err = errors.Join(err, q.stream.Close())
	}
	if q.conn != nil {
		err = errors.Join(err, q.conn.CloseWithError(0, ""))
	}
	return err
}
