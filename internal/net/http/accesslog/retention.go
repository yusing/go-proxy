package accesslog

import (
	"bufio"
	"bytes"
	"io"
	"strconv"
	"time"

	E "github.com/yusing/go-proxy/internal/error"
	"github.com/yusing/go-proxy/internal/utils/strutils"
)

type Retention struct {
	Days uint64 `json:"days"`
	Last uint64 `json:"last"`
}

const chunkSizeMax int64 = 128 * 1024 // 128KB

var (
	ErrInvalidSyntax = E.New("invalid syntax")
	ErrZeroValue     = E.New("zero value")
)

// Syntax:
//
// <N> days|weeks|months
//
// last <N>
//
// Parse implements strutils.Parser.
func (r *Retention) Parse(v string) (err error) {
	split := strutils.SplitSpace(v)
	if len(split) != 2 {
		return ErrInvalidSyntax.Subject(v)
	}
	switch split[0] {
	case "last":
		r.Last, err = strconv.ParseUint(split[1], 10, 64)
	default: // <N> days|weeks|months
		r.Days, err = strconv.ParseUint(split[0], 10, 64)
		if err != nil {
			return
		}
		switch split[1] {
		case "days":
		case "weeks":
			r.Days *= 7
		case "months":
			r.Days *= 30
		default:
			return ErrInvalidSyntax.Subject("unit " + split[1])
		}
	}
	if r.Days == 0 && r.Last == 0 {
		return ErrZeroValue
	}
	return
}

func (r *Retention) rotateLogFile(file AccessLogIO) (err error) {
	lastN := int(r.Last)
	days := int(r.Days)

	// Seek to end to get file size
	size, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}

	// Initialize ring buffer for last N lines
	lines := make([][]byte, 0, lastN|(days*1000))
	pos := size
	unprocessed := 0

	var chunk [chunkSizeMax]byte
	var lastLine []byte

	var shouldStop func() bool
	if days > 0 {
		cutoff := time.Now().AddDate(0, 0, -days)
		shouldStop = func() bool {
			return len(lastLine) > 0 && !parseLogTime(lastLine).After(cutoff)
		}
	} else {
		shouldStop = func() bool {
			return len(lines) == lastN
		}
	}

	// Read backwards until we have enough lines or reach start of file
	for pos > 0 {
		if pos > chunkSizeMax {
			pos -= chunkSizeMax
		} else {
			pos = 0
		}

		// Seek to the current chunk
		if _, err = file.Seek(pos, io.SeekStart); err != nil {
			return err
		}

		var nRead int
		// Read the chunk
		if nRead, err = file.Read(chunk[unprocessed:]); err != nil {
			return err
		}

		// last unprocessed bytes + read bytes
		curChunk := chunk[:unprocessed+nRead]
		unprocessed = len(curChunk)

		// Split into lines
		scanner := bufio.NewScanner(bytes.NewReader(curChunk))
		for !shouldStop() && scanner.Scan() {
			lastLine = scanner.Bytes()
			lines = append(lines, lastLine)
			unprocessed -= len(lastLine)
		}
		if shouldStop() {
			break
		}

		// move unprocessed bytes to the beginning for next iteration
		copy(chunk[:], curChunk[unprocessed:])
	}

	if days > 0 {
		// truncate to the end of the log within last N days
		return file.Truncate(pos)
	}

	// write lines to buffer in reverse order
	// since we read them backwards
	var buf bytes.Buffer
	for i := len(lines) - 1; i >= 0; i-- {
		buf.Write(lines[i])
		buf.WriteRune('\n')
	}

	return writeTruncate(file, &buf)
}

func writeTruncate(file AccessLogIO, buf *bytes.Buffer) (err error) {
	// Seek to beginning and truncate
	if _, err := file.Seek(0, 0); err != nil {
		return err
	}

	buffered := bufio.NewWriter(file)
	// Write buffer back to file
	nWritten, err := buffered.Write(buf.Bytes())
	if err != nil {
		return err
	}
	if err = buffered.Flush(); err != nil {
		return err
	}

	// Truncate file
	if err = file.Truncate(int64(nWritten)); err != nil {
		return err
	}

	// check bytes written == buffer size
	if nWritten != buf.Len() {
		return io.ErrShortWrite
	}
	return
}

func parseLogTime(line []byte) (t time.Time) {
	if len(line) == 0 {
		return
	}

	var start, end int
	const jsonStart = len(`{"time":"`)
	const jsonEnd = jsonStart + len(LogTimeFormat)

	if len(line) == '{' { // possibly json log
		start = jsonStart
		end = jsonEnd
	} else { // possibly common or combined format
		// Format: <virtual host> <host ip> - - [02/Jan/2006:15:04:05 -0700] ...
		start = bytes.IndexRune(line, '[')
		end = bytes.IndexRune(line[start+1:], ']')
		if start == -1 || end == -1 || start >= end {
			return
		}
	}

	timeStr := line[start+1 : end]
	t, _ = time.Parse(LogTimeFormat, string(timeStr)) // ignore error
	return
}
