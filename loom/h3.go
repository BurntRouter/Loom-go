package loom

import (
	"bufio"
	"context"
	"io"
	"net/http"

	"github.com/quic-go/quic-go/http3"
)

type h3Producer struct {
	pw *io.PipeWriter
	bw *bufio.Writer

	tr *http3.Transport

	done chan error
}

func newH3Producer(ctx context.Context, opt ClientOptions) (*h3Producer, error) {
	tlsCfg, err := tlsConfig(opt.TLS, []string{http3.NextProtoH3})
	if err != nil {
		return nil, err
	}
	tr := &http3.Transport{TLSClientConfig: tlsCfg}
	pr, pw := io.Pipe()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://"+opt.Addr+"/stream", pr)
	if err != nil {
		_ = tr.Close()
		return nil, err
	}
	cl := &http.Client{Transport: tr}

	p := &h3Producer{pw: pw, bw: bufio.NewWriter(pw), tr: tr, done: make(chan error, 1)}
	if err := WriteHello(p.bw, RoleProducer, opt.Name, opt.Room, opt.Token); err != nil {
		_ = pw.CloseWithError(err)
		_ = tr.Close()
		return nil, err
	}

	go func() {
		resp, err := cl.Do(req)
		if err != nil {
			p.done <- err
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode/100 != 2 {
			b, _ := io.ReadAll(resp.Body)
			p.done <- &httpError{Status: resp.Status, Body: string(b)}
			return
		}
		p.done <- nil
	}()

	return p, nil
}

func (p *h3Producer) Close() error {
	if p == nil {
		return nil
	}
	_ = p.bw.Flush()
	_ = p.pw.Close()
	err := <-p.done
	_ = p.tr.Close()
	return err
}

type h3Consumer struct {
	pw   *io.PipeWriter
	bw   *bufio.Writer
	resp *http.Response
	tr   *http3.Transport
}

func newH3Consumer(ctx context.Context, opt ClientOptions) (*h3Consumer, error) {
	tlsCfg, err := tlsConfig(opt.TLS, []string{http3.NextProtoH3})
	if err != nil {
		return nil, err
	}
	tr := &http3.Transport{TLSClientConfig: tlsCfg}
	pr, pw := io.Pipe()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://"+opt.Addr+"/stream", pr)
	if err != nil {
		_ = tr.Close()
		return nil, err
	}
	cl := &http.Client{Transport: tr}

	bw := bufio.NewWriter(pw)
	if err := WriteHello(bw, RoleConsumer, opt.Name, opt.Room, opt.Token); err != nil {
		_ = pw.CloseWithError(err)
		_ = tr.Close()
		return nil, err
	}
	if err := bw.Flush(); err != nil {
		_ = pw.CloseWithError(err)
		_ = tr.Close()
		return nil, err
	}

	resp, err := cl.Do(req)
	if err != nil {
		_ = pw.CloseWithError(err)
		_ = tr.Close()
		return nil, err
	}
	if resp.StatusCode/100 != 2 {
		b, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		_ = pw.Close()
		_ = tr.Close()
		return nil, &httpError{Status: resp.Status, Body: string(b)}
	}

	return &h3Consumer{pw: pw, bw: bw, resp: resp, tr: tr}, nil
}

func (c *h3Consumer) Close() error {
	if c == nil {
		return nil
	}
	_ = c.resp.Body.Close()
	_ = c.pw.Close()
	return c.tr.Close()
}

type httpError struct {
	Status string
	Body   string
}

func (e *httpError) Error() string { return "loom http error: " + e.Status + ": " + e.Body }
