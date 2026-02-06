package loom

import (
	"context"
	"io"
	"time"
)

func sleepCtx(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func (p *Producer) reconnectLoop(ctx context.Context) error {
	enabled, delay, max := p.opt.reconnect()
	if !enabled {
		return context.Canceled
	}
	attempts := 0
	for {
		_ = p.Close()
		np, err := NewProducer(ctx, p.opt)
		if err == nil {
			p.opt = np.opt
			p.q = np.q
			p.h3 = np.h3
			p.bw = np.bw
			return nil
		}
		attempts++
		if max > 0 && attempts >= max {
			return err
		}
		if err := sleepCtx(ctx, delay); err != nil {
			return err
		}
	}
}

func (c *Consumer) reconnectLoop(ctx context.Context) error {
	enabled, delay, max := c.opt.reconnect()
	if !enabled {
		return context.Canceled
	}
	attempts := 0
	for {
		_ = c.Close()
		nc, err := NewConsumer(ctx, c.opt)
		if err == nil {
			*c = *nc
			return nil
		}
		attempts++
		if max > 0 && attempts >= max {
			return err
		}
		if err := sleepCtx(ctx, delay); err != nil {
			return err
		}
	}
}

func rewindIfSeeker(r io.Reader) (io.Reader, func() bool) {
	rs, ok := r.(io.ReadSeeker)
	if !ok {
		return r, func() bool { return false }
	}
	pos, err := rs.Seek(0, io.SeekCurrent)
	if err != nil {
		return r, func() bool { return false }
	}
	return rs, func() bool {
		_, err := rs.Seek(pos, io.SeekStart)
		return err == nil
	}
}
