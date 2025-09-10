// Package parser provde a repeatedly read closer definition
package parser

import (
	"bytes"
	"context"
	"io"
)

// Reader stands for a generic IO reader interface
type Reader interface {
	Reset() (io.ReadCloser, error)    // Reset reset current reader to beginning if supported
	ReadAll() (buf []byte, err error) // ReadAll read all remain data in current reader
}

// RepeatedlySeekableReadCloser wraps an bytes.Reader
type RepeatedlySeekableReadCloser struct {
	*bytes.Reader
	body []byte
}

// NewRepeatableReadCloser will create a new RepeatedlySeekableReadCloser from io.ReadCloser and close the original one
func NewRepeatableReadCloser(rc io.ReadCloser) (io.ReadCloser, []byte, error) {
	buf, err := io.ReadAll(rc)
	if err != nil {
		return nil, nil, err
	}
	rc.Close()
	return RepeatedlySeekableReadCloser{Reader: bytes.NewReader(buf)}, buf, nil
}

// Close implement a nop-close method
func (r RepeatedlySeekableReadCloser) Close() error {
	r.Reset()
	return nil
}

// Reset will seek the read at the beginning
func (r RepeatedlySeekableReadCloser) Reset() (io.ReadCloser, error) {
	r.Reader.Seek(0, io.SeekStart)
	return r, nil
}

// ReadAll read all data into byte array
func (r RepeatedlySeekableReadCloser) ReadAll() (buf []byte, err error) {
	if r.body == nil {
		r.Reset()
		defer r.Close()
		buf, err = io.ReadAll(r.Reader)
		if err != nil {
			return
		}
		r.body = buf
	}
	return r.body, nil
}

// Encode encode content with given encoder
func (r RepeatedlySeekableReadCloser) Encode(ctx context.Context, encoder func(context.Context, []byte) (string, error)) (string, error) {
	buf, err := r.ReadAll()
	if err != nil {
		return "", err
	}
	return encoder(ctx, buf)
}
