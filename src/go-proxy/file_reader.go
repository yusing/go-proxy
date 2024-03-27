package main

import "os"

type Reader interface {
	Read() ([]byte, error)
}

type FileReader struct {
	Path string
}

func (r *FileReader) Read() ([]byte, error) {
	return os.ReadFile(r.Path)
}

type ByteReader struct {
	Data []byte
}

func (r *ByteReader) Read() ([]byte, error) {
	return r.Data, nil
}