package logger

import (
	"io"
)

type MultiWriterCloser struct {
	io.Writer
	closers []io.Closer
}

func NewMultiWriterCloser(writerClosers ...io.WriteCloser) *MultiWriterCloser {
	writers := make([]io.Writer, len(writerClosers))
	for i, w := range writerClosers {
		writers[i] = w
	}

	closers := make([]io.Closer, len(writerClosers))
	for i, w := range writerClosers {
		closers[i] = w
	}

	return &MultiWriterCloser{
		Writer:  io.MultiWriter(writers...),
		closers: closers,
	}
}

// Close closes all the underlying io.WriteCloser instances and returns the first error if any.
// If all close calls succeed, it returns nil.
func (m *MultiWriterCloser) Close() error {
	var firstErr error
	for _, closer := range m.closers {
		if closeErr := closer.Close(); closeErr != nil && firstErr == nil {
			firstErr = closeErr
		}
	}
	return firstErr
}
