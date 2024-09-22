package utils

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"syscall"

	E "github.com/yusing/go-proxy/error"
)

// TODO: move to "utils/io"
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
		r      ContextReader
		w      ContextWriter
		ctx    context.Context
		cancel context.CancelFunc
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
	_, cancel := context.WithCancel(ctx)
	return &Pipe{
		r:      ContextReader{ctx: ctx, Reader: r},
		w:      ContextWriter{ctx: ctx, Writer: w},
		ctx:    ctx,
		cancel: cancel,
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

func NewBidirectionalPipeIntermediate(ctx context.Context, listener io.ReadCloser, client io.ReadWriteCloser, target io.ReadWriteCloser) *BidirectionalPipe {
	return &BidirectionalPipe{
		pSrcDst: NewPipe(ctx, listener, client),
		pDstSrc: NewPipe(ctx, client, target),
	}
}

func (p BidirectionalPipe) Start() error {
	errCh := make(chan error, 2)
	go func() {
		errCh <- p.pSrcDst.Start()
	}()
	go func() {
		errCh <- p.pDstSrc.Start()
	}()
	return E.JoinE("bidirectional pipe error", <-errCh, <-errCh).Error()
}

func Copy(dst *ContextWriter, src *ContextReader) error {
	_, err := io.Copy(dst, src)
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
