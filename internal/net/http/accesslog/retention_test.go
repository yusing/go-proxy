package accesslog_test

import (
	"testing"
	"time"

	. "github.com/yusing/go-proxy/internal/net/http/accesslog"
	"github.com/yusing/go-proxy/internal/task"
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

func TestRetentionCommonFormat(t *testing.T) {
	var file MockFile
	logger := NewAccessLogger(task.RootTask("test", false), &file, &Config{
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

	// FIXME: keep days does not work
	t.Run("keep days", func(t *testing.T) {
		logger.Config().Retention = strutils.MustParse[*Retention]("3 days")
		ExpectEqual(t, logger.Config().Retention.Days, 3)
		ExpectEqual(t, logger.Config().Retention.Last, 0)
		ExpectNoError(t, logger.Rotate())
		ExpectEqual(t, file.Count(), 3)
	})
}
