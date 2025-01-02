package accesslog_test

import (
	"bytes"
	"io"
	"testing"
	"time"

	. "github.com/yusing/go-proxy/internal/net/http/accesslog"
	"github.com/yusing/go-proxy/internal/utils/strutils"
	. "github.com/yusing/go-proxy/internal/utils/testing"
)

func TestParseRetention(t *testing.T) {
	tests := []struct {
		input     string
		expected  *Retention
		shouldErr bool
	}{
		{"30 days", &Retention{Days: 30}, false},
		{"2 weeks", &Retention{Days: 14}, false},
		{"last 5", &Retention{Last: 5}, false},
		{"invalid input", &Retention{}, true},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			r := &Retention{}
			err := r.Parse(test.input)
			if !test.shouldErr {
				ExpectNoError(t, err)
			} else {
				ExpectDeepEqual(t, r, test.expected)
			}
		})
	}
}

type mockFile struct {
	data     []byte
	position int64
}

func (m *mockFile) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		m.position = offset
	case io.SeekCurrent:
		m.position += offset
	case io.SeekEnd:
		m.position = int64(len(m.data)) + offset
	}
	return m.position, nil
}

func (m *mockFile) Write(p []byte) (n int, err error) {
	m.data = append(m.data, p...)
	n = len(p)
	m.position += int64(n)
	return
}

func (m *mockFile) Name() string {
	return "mock"
}

func (m *mockFile) Read(p []byte) (n int, err error) {
	if m.position >= int64(len(m.data)) {
		return 0, io.EOF
	}
	n = copy(p, m.data[m.position:])
	m.position += int64(n)
	return n, nil
}

func (m *mockFile) ReadAt(p []byte, off int64) (n int, err error) {
	if off >= int64(len(m.data)) {
		return 0, io.EOF
	}
	n = copy(p, m.data[off:])
	m.position += int64(n)
	return n, nil
}

func (m *mockFile) Close() error {
	return nil
}

func (m *mockFile) Truncate(size int64) error {
	m.data = m.data[:size]
	m.position = size
	return nil
}

func (m *mockFile) Lock()   {}
func (m *mockFile) Unlock() {}

func (m *mockFile) Count() int {
	return bytes.Count(m.data[:m.position], []byte("\n"))
}

func (m *mockFile) Len() int64 {
	return m.position
}

func TestRetentionCommonFormat(t *testing.T) {
	file := mockFile{}
	logger := NewAccessLogger(nil, &file, &Config{
		Format:     FormatCommon,
		BufferSize: 1024,
	})
	for range 10 {
		logger.Log(req, resp)
	}
	logger.Flush(true)
	// test.Finish(nil)

	ExpectEqual(t, logger.Config().Retention, nil)
	ExpectTrue(t, file.Len() > 0)
	ExpectEqual(t, file.Count(), 10)

	t.Run("keep last", func(t *testing.T) {
		logger.Config().Retention = strutils.MustParse[*Retention]("last 5")
		ExpectEqual(t, logger.Config().Retention.Days, 0)
		ExpectEqual(t, logger.Config().Retention.Last, 5)
		ExpectNoError(t, logger.Rotate())
		ExpectEqual(t, file.Count(), 5)
	})

	_ = file.Truncate(0)

	timeNow := time.Now()
	for i := range 10 {
		logger.Formatter.(*CommonFormatter).GetTimeNow = func() time.Time {
			return timeNow.AddDate(0, 0, -i)
		}
		logger.Log(req, resp)
	}
	logger.Flush(true)

	t.Run("keep days", func(t *testing.T) {
		logger.Config().Retention = strutils.MustParse[*Retention]("3 days")
		ExpectEqual(t, logger.Config().Retention.Days, 3)
		ExpectEqual(t, logger.Config().Retention.Last, 0)
		ExpectNoError(t, logger.Rotate())
		ExpectEqual(t, file.Count(), 3)
	})
}
