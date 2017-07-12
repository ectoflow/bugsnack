package bugsnack

import (
	"context"
	"net/http"
	"os"
	"testing"

	"github.com/fromatob/bugsnack/error"
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

	er.Report(context.Background(), error.New("bugsnag test"))
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

	er.Report(context.Background(), error.New("bugsnag multireporter test"))
}
