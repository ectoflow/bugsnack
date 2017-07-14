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
	"github.com/fromatob/bugsnack/hashstruct"
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
	eventMetadata *hashstruct.Hash
}

func (er *BugsnagReporter) ReportWithMetadata(ctx context.Context, newErr interface{}, metadata *bugsnagMetadata) {
	payload := er.newPayload(error.New(newErr), metadata)

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
		er.Backup.Report(ctx, error.New("could not report to bugsnag"))
		return
	}
}

func (er *BugsnagReporter) Report(ctx context.Context, newErr interface{}) {
	er.ReportWithMetadata(ctx, newErr, &bugsnagMetadata{})
}

func (er *BugsnagReporter) newPayload(err *error.Error, metadata *bugsnagMetadata) *hashstruct.Hash {
	populateMetadata(metadata, err)

	return &hashstruct.Hash{
		"apiKey": er.APIKey,

		"notifier": &hashstruct.Hash{
			"name":    "Bugsnack/Bugsnag",
			"url":     "https://github.com/fromatob/bugsnack",
			"version": clientVersion,
		},

		"events": []*hashstruct.Hash{
			er.newEvent(err, metadata),
		},
	}
}

func (er *BugsnagReporter) newEvent(err *error.Error, metadata *bugsnagMetadata) *hashstruct.Hash {
	host, _ := os.Hostname()

	event := hashstruct.Hash{
		"PayloadVersion": "2",
		"exceptions": []*hashstruct.Hash{
			{
				"errorClass": metadata.errorClass,
				"message":    err.Error(),
				"stacktrace": formatStack(err.Stacktrace),
			},
		},
		"severity": metadata.severity,
		"app": &hashstruct.Hash{
			"releaseStage": er.ReleaseStage,
		},
		"device": &hashstruct.Hash{
			"hostname": host,
		},
	}

	if "" != metadata.groupingHash {
		event["groupingHash"] = metadata.groupingHash
	}

	if "" != metadata.context {
		event["context"] = metadata.context
	}

	if !metadata.eventMetadata.IsZeroInterface() {
		event["metaData"] = metadata.eventMetadata
	}

	return &event
}

func populateMetadata(metadata *bugsnagMetadata, err *error.Error) {
	if metadata.errorClass == "" {
		metadata.errorClass = reflect.TypeOf(err).String()
	}
	if metadata.severity == "" {
		metadata.severity = "error"
	}
}

func formatStack(s stack.CallStack) []hashstruct.Hash {
	var o []hashstruct.Hash

	for _, f := range s {
		line, _ := strconv.Atoi(fmt.Sprintf("%d", f))
		o = append(o, hashstruct.Hash{
			"method":     fmt.Sprintf("%n", f),
			"file":       fmt.Sprintf("%s", f),
			"lineNumber": line,
		})
	}

	return o
}
