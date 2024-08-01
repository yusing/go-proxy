package utils

import (
	"context"
	"io"
	"os"
	"sync/atomic"

	E "github.com/yusing/go-proxy/error"
)

type (
	Reader interface {
		Read() ([]byte, E.NestedError)
	}

	StdReader struct {
		r Reader
	}

	FileReader struct {
		Path string
	}

	ReadCloser struct {
		ctx    context.Context
		r      io.ReadCloser
		closed atomic.Bool
	}

	StdReadCloser struct {
		r *ReadCloser
	}

	ByteReader    []byte
	NewByteReader = ByteReader

	Pipe struct {
		r      ReadCloser
		w      io.WriteCloser
		ctx    context.Context
		cancel context.CancelFunc
	}

	BidirectionalPipe struct {
		pSrcDst Pipe
		pDstSrc Pipe
	}
)

func NewFileReader(path string) *FileReader {
	return &FileReader{Path: path}
}

func (r StdReader) Read() ([]byte, error) {
	return r.r.Read()
}

func (r *FileReader) Read() ([]byte, E.NestedError) {
	return E.Check(os.ReadFile(r.Path))
}

func (r ByteReader) Read() ([]byte, E.NestedError) {
	return r, E.Nil()
}

func (r *ReadCloser) Read(p []byte) (int, E.NestedError) {
	select {
	case <-r.ctx.Done():
		return 0, E.From(r.ctx.Err())
	default:
		return E.Check(r.r.Read(p))
	}
}

func (r *ReadCloser) Close() E.NestedError {
	if r.closed.Load() {
		return E.Nil()
	}
	r.closed.Store(true)
	return E.From(r.r.Close())
}

func (r StdReadCloser) Read(p []byte) (int, error) {
	return r.r.Read(p)
}

func (r StdReadCloser) Close() error {
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

func (p *Pipe) Start() E.NestedError {
	return Copy(p.ctx, p.w, &StdReadCloser{&p.r})
}

func (p *Pipe) Stop() E.NestedError {
	p.cancel()
	return E.Join("error stopping pipe", p.r.Close(), p.w.Close())
}

func (p *Pipe) Write(b []byte) (int, E.NestedError) {
	return E.Check(p.w.Write(b))
}

func NewBidirectionalPipe(ctx context.Context, rw1 io.ReadWriteCloser, rw2 io.ReadWriteCloser) *BidirectionalPipe {
	return &BidirectionalPipe{
		pSrcDst: *NewPipe(ctx, rw1, rw2),
		pDstSrc: *NewPipe(ctx, rw2, rw1),
	}
}

func NewBidirectionalPipeIntermediate(ctx context.Context, listener io.ReadCloser, client io.ReadWriteCloser, target io.ReadWriteCloser) *BidirectionalPipe {
	return &BidirectionalPipe{
		pSrcDst: *NewPipe(ctx, listener, client),
		pDstSrc: *NewPipe(ctx, client, target),
	}
}

func (p *BidirectionalPipe) Start() E.NestedError {
	errCh := make(chan E.NestedError, 2)
	go func() {
		errCh <- p.pSrcDst.Start()
	}()
	go func() {
		errCh <- p.pDstSrc.Start()
	}()
	for err := range errCh {
		if err.IsNotNil() {
			return err
		}
	}
	return E.Nil()
}

func (p *BidirectionalPipe) Stop() E.NestedError {
	return E.Join("error stopping pipe", p.pSrcDst.Stop(), p.pDstSrc.Stop())
}

func Copy(ctx context.Context, dst io.WriteCloser, src io.ReadCloser) E.NestedError {
	_, err := io.Copy(dst, StdReadCloser{&ReadCloser{ctx: ctx, r: src}})
	return E.From(err)
}