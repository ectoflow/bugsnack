package bugsnack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"strconv"

	"github.com/fromatob/bugsnack/error"
	"github.com/fromatob/bugsnack/internal/stack"
)

const clientVersion = "0.0.2"

// BugsnagReporter is an implementation of ErrorReporter that fires to
// BugSnag
type BugsnagReporter struct {
	Doer         Doer
	APIKey       string
	ReleaseStage string

	Backup ErrorReporter
}

func (er *BugsnagReporter) Report(ctx context.Context, newErr *error.Error) {
	payload := er.newPayload(newErr)

	var b bytes.Buffer
	err := json.NewEncoder(&b).Encode(payload)
	if err != nil {
		er.Backup.Report(ctx, error.New(err))
		return
	}

	req, err := http.NewRequest(http.MethodPost, "https://notify.bugsnag.com", &b)
	if err != nil {
		er.Backup.Report(ctx, error.New(err))
		return
	}
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")

	resp, err := er.Doer.Do(req)
	if err != nil {
		er.Backup.Report(ctx, error.New(err))
		return
	}
	defer func() {
		_, err = io.Copy(ioutil.Discard, io.LimitReader(resp.Body, 1024))
		if err != nil {
			er.Backup.Report(ctx, error.New(err))
		}
		err = resp.Body.Close()
		if err != nil {
			er.Backup.Report(ctx, error.New(err))
		}
	}()

	if resp.StatusCode != http.StatusOK {
		er.Backup.Report(ctx, error.New("could not report to bugsnag"))
		return
	}
}

func (er *BugsnagReporter) newPayload(err *error.Error) map[string]interface{} {
	host, _ := os.Hostname()
	return map[string]interface{}{
		"apiKey": er.APIKey,

		"notifier": map[string]interface{}{
			"name":    "Bugsnack/Bugsnag",
			"url":     "https://github.com/fromatob/bugsnack",
			"version": clientVersion,
		},

		"events": []map[string]interface{}{
			{
				"PayloadVersion": "2",
				"exceptions": []map[string]interface{}{
					{
						"errorClass": reflect.TypeOf(err).String(),
						"message":    err.Error(),
						"stacktrace": formatStack(err.Stacktrace),
					},
				},
				"severity": "error",
				"app": map[string]interface{}{
					"releaseStage": er.ReleaseStage,
				},
				"device": map[string]interface{}{
					"hostname": host,
				},
			},
		},
	}
}

func formatStack(s stack.CallStack) []map[string]interface{} {
	var o []map[string]interface{}

	for _, f := range s {
		line, _ := strconv.Atoi(fmt.Sprintf("%d", f))
		o = append(o, map[string]interface{}{
			"method":     fmt.Sprintf("%n", f),
			"file":       fmt.Sprintf("%s", f),
			"lineNumber": line,
		})
	}

	return o
}
