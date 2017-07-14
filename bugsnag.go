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
type bugsnagMetadata struct {
	errorClass    string
	context       string
	groupingHash  string
	severity      string
	eventMetadata *map[string]interface{}
}

func (metadata *bugsnagMetadata) populateMetadata(err *Error) {
	if metadata.errorClass == "" {
		metadata.errorClass = reflect.TypeOf(err).String()
	}
	if metadata.severity == "" {
		metadata.severity = "error"
	}
}

func (er *BugsnagReporter) ReportWithMetadata(ctx context.Context, newErr interface{}, metadata *bugsnagMetadata) {
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
	er.ReportWithMetadata(ctx, newErr, &bugsnagMetadata{})
}

func (er *BugsnagReporter) newPayload(err *Error, metadata *bugsnagMetadata) *map[string]interface{} {
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

func (er *BugsnagReporter) newEvent(err *Error, metadata *bugsnagMetadata) *map[string]interface{} {
	host, _ := os.Hostname()

	event := map[string]interface{}{
		"PayloadVersion": "2",
		"exceptions": []*map[string]interface{}{
			{
				"errorClass": metadata.errorClass,
				"message":    err.Error(),
				"stacktrace": formatStack(err.Stacktrace),
			},
		},
		"severity": metadata.severity,
		"app": &map[string]interface{}{
			"releaseStage": er.ReleaseStage,
		},
		"device": &map[string]interface{}{
			"hostname": host,
		},
	}

	if "" != metadata.groupingHash {
		event["groupingHash"] = metadata.groupingHash
	}

	if "" != metadata.context {
		event["context"] = metadata.context
	}

	if !IsZeroInterface(metadata.eventMetadata) {
		event["metaData"] = metadata.eventMetadata
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
