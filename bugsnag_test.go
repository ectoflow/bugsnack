package bugsnack

import (
	"context"
	"errors"
	"net/http"
	"os"
	"runtime"
	"testing"

	"github.com/fromatob/bugsnack/hashstruct"
)

func TestErrorReporter(t *testing.T) {
	if os.Getenv("BUGSNAG_TEST") != "T" {
		t.Skip("not running bugsnag reporter test")
	}

	er := &BugsnagReporter{
		APIKey:       os.Getenv("BUGSNAG_API_KEY"),
		Doer:         http.DefaultClient,
		ReleaseStage: "development",
		Backup:       nil,
	}

	er.Report(context.Background(), errors.New("bugsnag error test"))

	er.ReportWithMetadata(context.Background(), NewError("bugsnag test"), &bugsnagMetadata{
		groupingHash: "net.timeout",
		eventMetadata: &hashstruct.Hash{
			"data": hashstruct.Hash{
				"os": runtime.GOOS,
			},
			"key1": "value1",
			"key2": "value2",
			"arbitraryData": hashstruct.Hash{
				"goVersion": runtime.Version(),
				"nested": hashstruct.Hash{
					"nestedKey": "value",
				},
			},
		},
	})
}

func TestNestedErrorReporter(t *testing.T) {
	if os.Getenv("BUGSNAG_TEST") != "T" {
		t.Skip("not running bugsnag reporter test")
	}

	er := MultiReporter{
		Reporters: []ErrorReporter{&BugsnagReporter{
			APIKey:       os.Getenv("BUGSNAG_API_KEY"),
			Doer:         http.DefaultClient,
			ReleaseStage: "development",
			Backup:       nil,
		}}}

	er.Report(context.Background(), NewError("bugsnag multireporter test"))
}
