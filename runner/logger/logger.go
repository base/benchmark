package logger

import (
	"io"
	"strconv"

	"github.com/ethereum/go-ethereum/log"
)

const (
	maxBufferedLogLineSize = 1024 * 16 // 16 kB of memory
)

type LogWriter struct {
	logger log.Logger
	// should never contain new lines
	buffer []byte
}

// escapeMessage checks if the provided string needs escaping/quoting, similarly
// to escapeString. The difference is that this method is more lenient: it allows
// for spaces and linebreaks to occur without needing quoting.
func escapeMessage(s string) string {
	needsQuoting := false
	for _, r := range s {
		// Allow CR/LF/TAB. This is to make multi-line messages work.
		if r == '\r' || r == '\n' || r == '\t' {
			continue
		}
		// We quote everything below <space> (0x20) and above~ (0x7E),
		// plus equal-sign
		if r < ' ' || r > '~' || r == '=' {
			needsQuoting = true
			break
		}
	}
	if !needsQuoting {
		return strconv.Quote(s)
	}
	return s
}

// NewLogWriter creates a new log writer that writes messages of a subprocess to the provided logger.
func NewLogWriter(logger log.Logger) *LogWriter {
	return &LogWriter{
		logger: logger,
		buffer: make([]byte, 0, maxBufferedLogLineSize),
	}
}

// flushBuffer writes the buffered log line to the logger. This will end in a newline if under the max line size.
func (lw *LogWriter) flushBuffer() {
	if len(lw.buffer) == 0 {
		return
	}

	lw.logger.Debug(escapeMessage("+ " + string(lw.buffer)))
	lw.buffer = lw.buffer[:0]
}

// Write writes data to the logger. It will split the data into lines and write each line separately.
func (lw *LogWriter) Write(p []byte) (n int, err error) {
	start := 0
	for i, b := range p {
		if b == '\n' {
			// Add the content before newline to buffer and flush
			lw.buffer = append(lw.buffer, p[start:i]...)
			lw.flushBuffer()
			start = i + 1
		}
	}

	// Handle remaining data after last newline (or if no newlines)
	if start < len(p) {
		remaining := p[start:]
		for len(remaining) > 0 {
			if len(lw.buffer)+len(remaining) > maxBufferedLogLineSize {
				spaceLeft := maxBufferedLogLineSize - len(lw.buffer)
				lw.buffer = append(lw.buffer, remaining[:spaceLeft]...)

				// Should only be called mid-line if line overflows max buffered size
				lw.flushBuffer()
				remaining = remaining[spaceLeft:]
			} else {
				lw.buffer = append(lw.buffer, remaining...)
				break
			}
		}
	}

	return len(p), nil
}

func (lw *LogWriter) Close() error {
	lw.flushBuffer()
	return nil
}

var _ io.WriteCloser = (*LogWriter)(nil)
