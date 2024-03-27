package main

import (
	"context"
	"io"
	"sync"
)

type ReadCloser struct {
	ctx context.Context
	r   io.ReadCloser
}

func (r *ReadCloser) Read(p []byte) (int, error) {
	select {
	case <-r.ctx.Done():
		return 0, r.ctx.Err()
	default:
		return r.r.Read(p)
	}
}

func (r *ReadCloser) Close() error {
	return r.r.Close()
}

type Pipe struct {
	r      ReadCloser
	w      io.WriteCloser
	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
}

func NewPipe(ctx context.Context, r io.ReadCloser, w io.WriteCloser) *Pipe {
	ctx, cancel := context.WithCancel(ctx)
	return &Pipe{
		r:      ReadCloser{ctx, r},
		w:      w,
		ctx:    ctx,
		cancel: cancel,
	}
}

func (p *Pipe) Start() {
	p.wg.Add(1)
	go func() {
		Copy(p.ctx, p.w, &p.r)
		p.wg.Done()
	}()
}

func (p *Pipe) Stop() {
	p.cancel()
	p.wg.Wait()
}

func (p *Pipe) Close() (error, error) {
	return p.r.Close(), p.w.Close()
}

func (p *Pipe) Wait() {
	p.wg.Wait()
}

type BidirectionalPipe struct {
	pSrcDst Pipe
	pDstSrc Pipe
}

func NewBidirectionalPipe(ctx context.Context, rw1 io.ReadWriteCloser, rw2 io.ReadWriteCloser) *BidirectionalPipe {
	return &BidirectionalPipe{
		pSrcDst: *NewPipe(ctx, rw1, rw2),
		pDstSrc: *NewPipe(ctx, rw2, rw1),
	}
}

func (p *BidirectionalPipe) Start() {
	p.pSrcDst.Start()
	p.pDstSrc.Start()
}

func (p *BidirectionalPipe) Stop() {
	p.pSrcDst.Stop()
	p.pDstSrc.Stop()
}

func (p *BidirectionalPipe) Close() (error, error) {
	return p.pSrcDst.Close()
}

func (p *BidirectionalPipe) Wait() {
	p.pSrcDst.Wait()
	p.pDstSrc.Wait()
}

func Copy(ctx context.Context, dst io.WriteCloser, src io.ReadCloser) error {
	_, err := io.Copy(dst, &ReadCloser{ctx, src})
	return err
}
