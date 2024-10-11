package utils

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"sync"
	"syscall"

	E "github.com/yusing/go-proxy/internal/error"
)

// TODO: move to "utils/io".
type (
	FileReader struct {
		Path string
	}

	ContextReader struct {
		ctx context.Context
		io.Reader
	}

	ContextWriter struct {
		ctx context.Context
		io.Writer
	}

	Pipe struct {
		r ContextReader
		w ContextWriter
	}

	BidirectionalPipe struct {
		pSrcDst *Pipe
		pDstSrc *Pipe
	}
)

func (r *ContextReader) Read(p []byte) (int, error) {
	select {
	case <-r.ctx.Done():
		return 0, r.ctx.Err()
	default:
		return r.Reader.Read(p)
	}
}

func (w *ContextWriter) Write(p []byte) (int, error) {
	select {
	case <-w.ctx.Done():
		return 0, w.ctx.Err()
	default:
		return w.Writer.Write(p)
	}
}

func NewPipe(ctx context.Context, r io.ReadCloser, w io.WriteCloser) *Pipe {
	return &Pipe{
		r: ContextReader{ctx: ctx, Reader: r},
		w: ContextWriter{ctx: ctx, Writer: w},
	}
}

func (p *Pipe) Start() (err error) {
	err = Copy(&p.w, &p.r)
	switch {
	case
		// NOTE: ignoring broken pipe and connection reset by peer
		errors.Is(err, syscall.EPIPE),
		errors.Is(err, syscall.ECONNRESET):
		return nil
	}
	return err
}

func NewBidirectionalPipe(ctx context.Context, rw1 io.ReadWriteCloser, rw2 io.ReadWriteCloser) BidirectionalPipe {
	return BidirectionalPipe{
		pSrcDst: NewPipe(ctx, rw1, rw2),
		pDstSrc: NewPipe(ctx, rw2, rw1),
	}
}

func (p BidirectionalPipe) Start() error {
	var wg sync.WaitGroup
	wg.Add(2)
	b := E.NewBuilder("bidirectional pipe error")
	go func() {
		b.AddE(p.pSrcDst.Start())
		wg.Done()
	}()
	go func() {
		b.AddE(p.pDstSrc.Start())
		wg.Done()
	}()
	wg.Wait()
	return b.Build().Error()
}

func Copy(dst *ContextWriter, src *ContextReader) error {
	_, err := io.Copy(dst, src)
	return err
}

func Copy2(ctx context.Context, dst io.Writer, src io.Reader) error {
	return Copy(&ContextWriter{ctx: ctx, Writer: dst}, &ContextReader{ctx: ctx, Reader: src})
}

func LoadJSON[T any](path string, pointer *T) E.NestedError {
	data, err := E.Check(os.ReadFile(path))
	if err.HasError() {
		return err
	}
	return E.From(json.Unmarshal(data, pointer))
}

func SaveJSON[T any](path string, pointer *T, perm os.FileMode) E.NestedError {
	data, err := E.Check(json.Marshal(pointer))
	if err.HasError() {
		return err
	}
	return E.From(os.WriteFile(path, data, perm))
}
