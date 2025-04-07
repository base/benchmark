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
	var err error
	for _, closer := range m.closers {
		if err != nil {
			err = closer.Close()
		} else {
			_ = closer.Close()
		}
	}
	return nil
}
