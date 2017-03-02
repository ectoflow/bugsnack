package bugsnack

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"strconv"

	"github.com/fromatob/bugsnack/internal/stack"
)

// BugsnagReporter is an implementation of ErrorReporter that fires to
// BugSnag
type BugsnagReporter struct {
	Doer         Doer
	APIKey       string
	ReleaseStage string

	Backup ErrorReporter
}

func (er *BugsnagReporter) Report(ctx context.Context, newErr error) {
	var b bytes.Buffer
	err := json.NewEncoder(&b).Encode(er.newPayload(newErr))
	if err != nil {
		er.Backup.Report(ctx, err)
		return
	}

	req, err := http.NewRequest(http.MethodPost, "https://notify.bugsnag.com", &b)
	if err != nil {
		er.Backup.Report(ctx, err)
		return
	}
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")

	resp, err := er.Doer.Do(req)
	if err != nil {
		er.Backup.Report(ctx, err)
		return
	}
	defer func() {
		_, err = io.Copy(ioutil.Discard, io.LimitReader(resp.Body, 1024))
		if err != nil {
			er.Backup.Report(ctx, err)
		}
		err = resp.Body.Close()
		if err != nil {
			er.Backup.Report(ctx, err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		er.Backup.Report(ctx, errors.New("could not report to bugsnag"))
		return
	}
}

func (er *BugsnagReporter) newPayload(err error) map[string]interface{} {
	c := stack.Trace()

	host, _ := os.Hostname()
	return map[string]interface{}{
		"apiKey": er.APIKey,

		"notifier": map[string]interface{}{
			"name":    "Bugsnack/Bugsnag",
			"url":     "https://github.com/fromatob/bugsnack",
			"version": "0.0.1",
		},

		"events": []map[string]interface{}{
			{
				"PayloadVersion": "2",
				"exceptions": []map[string]interface{}{
					{
						"errorClass": stack.Caller(2).String(),
						"message":    fmt.Sprint(err),
						"stacktrace": formatStack(c),
					},
				},
				"severity": "error",
				"app": map[string]interface{}{
					"releaseStage": er.ReleaseStage,
				},
				"device": map[string]interface{}{
					"hostname": host,
				},
				"context": fmt.Sprint(reflect.TypeOf(err)),
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
