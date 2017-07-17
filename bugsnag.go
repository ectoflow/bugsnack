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
type BugsnagMetadata struct {
	ErrorClass    string
	Context       string
	GroupingHash  string
	Severity      string
	EventMetadata *map[string]interface{}
}

func (metadata *BugsnagMetadata) populateMetadata(err *Error) {
	if metadata.ErrorClass == "" {
		metadata.ErrorClass = reflect.TypeOf(err).String()
	}
	if metadata.Severity == "" {
		metadata.Severity = "error"
	}
}

func (er *BugsnagReporter) ReportWithMetadata(ctx context.Context, newErr interface{}, metadata *BugsnagMetadata) {
	payload := er.newPayload(NewError(newErr), metadata)

	var b bytes.Buffer
	err := json.NewEncoder(&b).Encode(payload)
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
		er.Backup.Report(ctx, NewError("could not report to bugsnag"))
		return
	}
}

func (er *BugsnagReporter) Report(ctx context.Context, newErr interface{}) {
	er.ReportWithMetadata(ctx, newErr, &BugsnagMetadata{})
}

func (er *BugsnagReporter) newPayload(err *Error, metadata *BugsnagMetadata) *map[string]interface{} {
	metadata.populateMetadata(err)

	return &map[string]interface{}{
		"apiKey": er.APIKey,

		"notifier": &map[string]interface{}{
			"name":    "Bugsnack/Bugsnag",
			"url":     "https://github.com/fromatob/bugsnack",
			"version": clientVersion,
		},

		"events": []*map[string]interface{}{
			er.newEvent(err, metadata),
		},
	}
}

func (er *BugsnagReporter) newEvent(err *Error, metadata *BugsnagMetadata) *map[string]interface{} {
	host, _ := os.Hostname()

	event := map[string]interface{}{
		"PayloadVersion": "2",
		"exceptions": []*map[string]interface{}{
			{
				"errorClass": metadata.ErrorClass,
				"message":    err.Error(),
				"stacktrace": formatStack(err.Stacktrace),
			},
		},
		"severity": metadata.Severity,
		"app": &map[string]interface{}{
			"releaseStage": er.ReleaseStage,
		},
		"device": &map[string]interface{}{
			"hostname": host,
		},
	}

	if "" != metadata.GroupingHash {
		event["groupingHash"] = metadata.GroupingHash
	}

	if "" != metadata.Context {
		event["context"] = metadata.Context
	}

	if !IsZeroInterface(metadata.EventMetadata) {
		event["metaData"] = metadata.EventMetadata
	}

	return &event
}

func IsZeroInterface(i interface{}) bool {
	return i == reflect.Zero(reflect.TypeOf(i)).Interface()
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
