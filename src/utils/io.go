package utils

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"sync/atomic"

	E "github.com/yusing/go-proxy/error"
)

// TODO: move to "utils/io"
type (
	FileReader struct {
		Path string
	}

	ReadCloser struct {
		ctx    context.Context
		r      io.ReadCloser
		closed atomic.Bool
	}

	Pipe struct {
		r      ReadCloser
		w      io.WriteCloser
		ctx    context.Context
		cancel context.CancelFunc
	}

	BidirectionalPipe struct {
		pSrcDst *Pipe
		pDstSrc *Pipe
	}
)

func (r *ReadCloser) Read(p []byte) (int, error) {
	select {
	case <-r.ctx.Done():
		return 0, r.ctx.Err()
	default:
		return r.r.Read(p)
	}
}

func (r *ReadCloser) Close() error {
	if r.closed.Load() {
		return nil
	}
	r.closed.Store(true)
	return r.r.Close()
}

func NewPipe(ctx context.Context, r io.ReadCloser, w io.WriteCloser) *Pipe {
	ctx, cancel := context.WithCancel(ctx)
	return &Pipe{
		r:      ReadCloser{ctx: ctx, r: r},
		w:      w,
		ctx:    ctx,
		cancel: cancel,
	}
}

func (p *Pipe) Start() error {
	return Copy(p.ctx, p.w, &p.r)
}

func (p *Pipe) Stop() error {
	p.cancel()
	return E.JoinE("error stopping pipe", p.r.Close(), p.w.Close()).Error()
}

func (p *Pipe) Write(b []byte) (int, error) {
	return p.w.Write(b)
}

func NewBidirectionalPipe(ctx context.Context, rw1 io.ReadWriteCloser, rw2 io.ReadWriteCloser) *BidirectionalPipe {
	return &BidirectionalPipe{
		pSrcDst: NewPipe(ctx, rw1, rw2),
		pDstSrc: NewPipe(ctx, rw2, rw1),
	}
}

func NewBidirectionalPipeIntermediate(ctx context.Context, listener io.ReadCloser, client io.ReadWriteCloser, target io.ReadWriteCloser) *BidirectionalPipe {
	return &BidirectionalPipe{
		pSrcDst: NewPipe(ctx, listener, client),
		pDstSrc: NewPipe(ctx, client, target),
	}
}

func (p *BidirectionalPipe) Start() error {
	errCh := make(chan error, 2)
	go func() {
		errCh <- p.pSrcDst.Start()
	}()
	go func() {
		errCh <- p.pDstSrc.Start()
	}()
	for err := range errCh {
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *BidirectionalPipe) Stop() error {
	return E.JoinE("error stopping pipe", p.pSrcDst.Stop(), p.pDstSrc.Stop()).Error()
}

func Copy(ctx context.Context, dst io.WriteCloser, src io.ReadCloser) error {
	_, err := io.Copy(dst, &ReadCloser{ctx: ctx, r: src})
	return err
}

func LoadJson[T any](path string, pointer *T) E.NestedError {
	data, err := E.Check(os.ReadFile(path))
	if err.HasError() {
		return err
	}
	return E.From(json.Unmarshal(data, pointer))
}

func SaveJson[T any](path string, pointer *T, perm os.FileMode) E.NestedError {
	data, err := E.Check(json.Marshal(pointer))
	if err.HasError() {
		return err
	}
	return E.From(os.WriteFile(path, data, perm))
}
